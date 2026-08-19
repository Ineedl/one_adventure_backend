// Package kafka provides shared Kafka producers and consumer-group consumers.
//
// MessageHandler controls offset submission by calling CommitFunc after its
// business operation succeeds. If it does not call CommitFunc, Run does not
// commit that message. Handlers should be idempotent because Kafka messages can
// be delivered more than once.
package kafka
