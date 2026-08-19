local stock = tonumber(redis.call('HGET', KEYS[1], 'stock'))
if not stock then
    return -1
end

local purchase_num = tonumber(ARGV[1])
if purchase_num <= 0 or stock < purchase_num then
    return -2
end

local limit_num = tonumber(redis.call('HGET', KEYS[1], 'limit_num') or '0')
local purchased_num = tonumber(redis.call('GET', KEYS[2]) or '0')
if limit_num > 0 and purchased_num + purchase_num > limit_num then
    return -3
end

redis.call('HINCRBY', KEYS[1], 'stock', -purchase_num)
redis.call('INCRBY', KEYS[2], purchase_num)

local ttl = redis.call('TTL', KEYS[1])
if ttl > 0 then
    redis.call('EXPIRE', KEYS[2], ttl)
end

return stock - purchase_num
