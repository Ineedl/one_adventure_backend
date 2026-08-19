package promotion

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	obslog "one_adventure_observability_log"
	tracekit "one_adventure_observability_trace/trace"
	pb "one_adventure_rpc/proto/promotion"
	contractevent "one_adventure_servicekit/api-contract/event"
	kafkakit "one_adventure_servicekit/kafka"
	"one_adventure_servicekit/token"
	"promotion/internal/dao"
	"promotion/internal/model/entity"
	promotionscript "promotion/script"
)

type service struct {
	pb.UnimplementedPromotionServiceServer
	producer *kafkakit.Producer
}

func newService(p *kafkakit.Producer) pb.PromotionServiceServer { return &service{producer: p} }

func (s *service) Seckill(ctx context.Context, r *pb.SeckillReq) (*pb.SeckillResp, error) {
	requestID := r.GetRequestId()
	if requestID == "" {
		requestID = uuid.NewString()
	}
	ctx = tracekit.WithRequestID(ctx, requestID)
	ctx, span := tracekit.Tracer().Start(ctx, "promotion.seckill", oteltrace.WithAttributes(
		attribute.String("request.id", tracekit.RequestID(ctx)),
		attribute.Int64("promotion.id", int64(r.GetPromotionId())),
		attribute.Int64("product.id", int64(r.GetProductId())),
	))
	defer span.End()
	u, ok := token.UserInfoFromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user info is required")
	}
	if r.PromotionId == 0 || r.ProductId == 0 || r.PayNum <= 0 || r.Price < 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid seckill request")
	}
	stockKey := fmt.Sprintf("promotion:%d:%d", r.PromotionId, r.ProductId)
	limitKey := fmt.Sprintf("promotion_limit:%d:%d:%d", r.PromotionId, r.ProductId, u.ID)
	v, err := g.Redis().GroupScript().Eval(ctx, promotionscript.Seckill, 2, []string{stockKey, limitKey}, []any{r.PayNum})
	if err != nil {
		return nil, status.Error(codes.Internal, "execute seckill failed")
	}
	switch v.Int() {
	case -1:
		return nil, status.Error(codes.FailedPrecondition, "promotion stock is not loaded")
	case -2:
		return nil, status.Error(codes.ResourceExhausted, "promotion stock is insufficient")
	case -3:
		return nil, status.Error(codes.FailedPrecondition, "promotion purchase limit exceeded")
	}
	event := contractevent.PromotionOrderCreate{
		RequestID:    tracekit.RequestID(ctx),
		TraceID:      tracekit.TraceID(ctx),
		PromotionID:  r.PromotionId,
		ProductID:    r.ProductId,
		UserID:       u.ID,
		PayNum:       r.PayNum,
		Price:        r.Price,
		CurrencyType: r.CurrencyType,
		CreatedAt:    time.Now().UnixMilli(),
	}
	b, _ := json.Marshal(event)
	if err = s.producer.Write(ctx, contractevent.PromotionOrderCreateTopic, strconv.FormatUint(u.ID, 10), b); err != nil {
		compensateCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		compensateResult, compensateErr := g.Redis().GroupScript().Eval(
			compensateCtx,
			promotionscript.SeckillCompensate,
			2,
			[]string{stockKey, limitKey},
			[]any{r.PayNum},
		)
		if compensateErr != nil || compensateResult.Int() < 0 {
			obslog.Error(ctx, "compensate promotion seckill after kafka failure", map[string]any{
				"promotion_id":      r.PromotionId,
				"product_id":        r.ProductId,
				"user_id":           u.ID,
				"pay_num":           r.PayNum,
				"kafka_error":       err.Error(),
				"compensate_error":  compensateErr,
				"compensate_result": compensateResult.Int(),
			})
			return nil, status.Error(codes.Internal, "publish promotion order failed and stock compensation failed")
		}
		return nil, status.Error(codes.Unavailable, "publish promotion order failed")
	}
	return &pb.SeckillResp{Success: true}, nil
}

func (s *service) PromotionStockRefresh(ctx context.Context, r *pb.PromotionStockRefreshReq) (*pb.PromotionStockRefreshResp, error) {
	p := dao.Promotion.Columns()
	pp := dao.PromotionProduct.Columns()
	m := dao.PromotionProduct.Ctx(ctx).As("pp").LeftJoin(dao.Promotion.Table()+" p", "p."+p.PromotionId+"=pp."+pp.PromotionId).Where("p."+p.Status, 1).WhereLTE("p."+p.StartTime, time.Now()).WhereGT("p."+p.EndTime, time.Now())
	if r.PromotionId != nil {
		m = m.Where("pp."+pp.PromotionId, r.GetPromotionId())
	}
	if r.ProductId != nil {
		if r.PromotionId == nil {
			return nil, status.Error(codes.InvalidArgument, "promotion_id is required when product_id is set")
		}
		m = m.Where("pp."+pp.ProductId, r.GetProductId())
	}
	var rows []struct {
		entity.PromotionProduct
		EndTime time.Time `json:"endTime"`
	}
	if err := m.Fields("pp.*", "p."+p.EndTime+" end_time").Scan(&rows); err != nil {
		return nil, status.Error(codes.Internal, "query promotion stock failed")
	}
	for _, row := range rows {
		key := fmt.Sprintf("promotion:%d:%d", row.PromotionId, row.ProductId)
		ttl := time.Until(row.EndTime)
		if ttl <= 0 {
			continue
		}
		_, err := g.Redis().HSet(ctx, key, map[string]any{"stock": row.Stock, "price": row.Price, "currency_type": row.CurrencyType, "limit_num": func() int {
			if row.LimitType == 1 {
				return row.LimitNum
			}
			return 0
		}()})
		if err == nil {
			_, err = g.Redis().Expire(ctx, key, int64(ttl.Seconds()))
		}
		if err != nil {
			return nil, status.Error(codes.Internal, "write promotion stock failed")
		}
	}
	return &pb.PromotionStockRefreshResp{RefreshedCount: int32(len(rows))}, nil
}
