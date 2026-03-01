package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const (
	DefaultBullPrefix = "bull"

	DataGatheringQueueName        = "DATA_GATHERING_QUEUE"
	PortfolioSnapshotQueueName    = "PORTFOLIO_SNAPSHOT_COMPUTATION_QUEUE"
	JobGatherAssetProfile         = "GATHER_ASSET_PROFILE"
	JobGatherHistoricalMarketData = "GATHER_HISTORICAL_MARKET_DATA"
	JobComputePortfolioSnapshot   = "COMPUTE_PORTFOLIO_SNAPSHOT"
)

func DefaultQueues() []string {
	return []string{DataGatheringQueueName, PortfolioSnapshotQueueName}
}

type AdminJob struct {
	AttemptsMade int            `json:"attemptsMade"`
	Data         map[string]any `json:"data"`
	FinishedOn   int64          `json:"finishedOn"`
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Opts         map[string]any `json:"opts"`
	Stacktrace   string         `json:"stacktrace"`
	State        string         `json:"state"`
	Timestamp    int64          `json:"timestamp"`
}

type Service struct {
	rdb    *goredis.Client
	prefix string
	queues []string
}

func NewService(rdb *goredis.Client, prefix string, queues []string) *Service {
	if prefix == "" {
		prefix = DefaultBullPrefix
	}
	if len(queues) == 0 {
		queues = DefaultQueues()
	}
	return &Service{rdb: rdb, prefix: prefix, queues: queues}
}

func (s *Service) Enqueue(ctx context.Context, queueName, jobName string, data map[string]any) (string, error) {
	if s == nil || s.rdb == nil {
		return "", nil
	}
	if queueName == "" {
		queueName = DataGatheringQueueName
	}
	idKey := s.key(queueName, "id")
	jobID, err := s.rdb.Incr(ctx, idKey).Result()
	if err != nil {
		return "", fmt.Errorf("incr job id: %w", err)
	}

	payload, _ := json.Marshal(data)
	now := time.Now().UnixMilli()
	jobIDStr := strconv.FormatInt(jobID, 10)

	jobKey := s.key(queueName, jobIDStr)
	fields := map[string]any{
		"name":         jobName,
		"data":         string(payload),
		"opts":         "{}",
		"timestamp":    strconv.FormatInt(now, 10),
		"delay":        "0",
		"priority":     "0",
		"attempts":     "1",
		"attemptsMade": "0",
	}

	pipe := s.rdb.TxPipeline()
	pipe.HSet(ctx, jobKey, fields)
	pipe.LPush(ctx, s.key(queueName, "wait"), jobIDStr)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return "", fmt.Errorf("enqueue job: %w", err)
	}
	return jobIDStr, nil
}

func (s *Service) ListJobs(ctx context.Context, limit int64) ([]AdminJob, error) {
	if s == nil || s.rdb == nil {
		return []AdminJob{}, nil
	}
	if limit <= 0 {
		limit = 200
	}

	type queueJob struct {
		queue string
		id    string
		state string
	}
	seen := map[string]bool{}
	var refs []queueJob

	for _, queueName := range s.queues {
		waitIDs, _ := s.rdb.LRange(ctx, s.key(queueName, "wait"), 0, limit-1).Result()
		activeIDs, _ := s.rdb.LRange(ctx, s.key(queueName, "active"), 0, limit-1).Result()
		delayedIDs, _ := s.rdb.ZRevRange(ctx, s.key(queueName, "delayed"), 0, limit-1).Result()
		completedIDs, _ := s.rdb.ZRevRange(ctx, s.key(queueName, "completed"), 0, limit-1).Result()
		failedIDs, _ := s.rdb.ZRevRange(ctx, s.key(queueName, "failed"), 0, limit-1).Result()

		appendIDs := func(ids []string, state string) {
			for _, id := range ids {
				refKey := queueName + ":" + id
				if seen[refKey] {
					continue
				}
				seen[refKey] = true
				refs = append(refs, queueJob{queue: queueName, id: id, state: state})
			}
		}
		appendIDs(waitIDs, "waiting")
		appendIDs(activeIDs, "active")
		appendIDs(delayedIDs, "delayed")
		appendIDs(completedIDs, "completed")
		appendIDs(failedIDs, "failed")
	}

	jobs := make([]AdminJob, 0, len(refs))
	for _, ref := range refs {
		raw, err := s.rdb.HGetAll(ctx, s.key(ref.queue, ref.id)).Result()
		if err != nil || len(raw) == 0 {
			continue
		}

		var data map[string]any
		if v := raw["data"]; v != "" {
			_ = json.Unmarshal([]byte(v), &data)
		}
		if data == nil {
			data = map[string]any{}
		}

		var opts map[string]any
		if v := raw["opts"]; v != "" {
			_ = json.Unmarshal([]byte(v), &opts)
		}
		if opts == nil {
			opts = map[string]any{}
		}

		attemptsMade, _ := strconv.Atoi(raw["attemptsMade"])
		finishedOn, _ := strconv.ParseInt(raw["finishedOn"], 10, 64)
		timestamp, _ := strconv.ParseInt(raw["timestamp"], 10, 64)

		stacktrace := raw["stacktrace"]
		if stacktrace == "" {
			stacktrace = raw["failedReason"]
		}

		jobs = append(jobs, AdminJob{
			AttemptsMade: attemptsMade,
			Data:         data,
			FinishedOn:   finishedOn,
			ID:           ref.id,
			Name:         raw["name"],
			Opts:         opts,
			Stacktrace:   stacktrace,
			State:        ref.state,
			Timestamp:    timestamp,
		})
	}

	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].Timestamp > jobs[j].Timestamp
	})

	return jobs, nil
}

func (s *Service) DeleteJob(ctx context.Context, id string) error {
	if s == nil || s.rdb == nil || strings.TrimSpace(id) == "" {
		return nil
	}
	for _, queueName := range s.queues {
		if err := s.deleteJobFromQueue(ctx, queueName, id); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) DeleteAllJobs(ctx context.Context) error {
	if s == nil || s.rdb == nil {
		return nil
	}
	jobs, err := s.ListJobs(ctx, 1000)
	if err != nil {
		return err
	}
	for _, queueName := range s.queues {
		for _, job := range jobs {
			_ = s.deleteJobFromQueue(ctx, queueName, job.ID)
		}
	}
	return nil
}

func (s *Service) ExecuteJob(ctx context.Context, id string) error {
	if s == nil || s.rdb == nil || strings.TrimSpace(id) == "" {
		return nil
	}
	for _, queueName := range s.queues {
		jobKey := s.key(queueName, id)
		exists, err := s.rdb.Exists(ctx, jobKey).Result()
		if err != nil || exists == 0 {
			continue
		}
		pipe := s.rdb.TxPipeline()
		pipe.ZRem(ctx, s.key(queueName, "failed"), id)
		pipe.ZRem(ctx, s.key(queueName, "completed"), id)
		pipe.ZRem(ctx, s.key(queueName, "delayed"), id)
		pipe.LRem(ctx, s.key(queueName, "active"), 0, id)
		pipe.LRem(ctx, s.key(queueName, "wait"), 0, id)
		pipe.LPush(ctx, s.key(queueName, "wait"), id)
		_, err = pipe.Exec(ctx)
		if err != nil {
			return fmt.Errorf("execute job: %w", err)
		}
		return nil
	}
	return nil
}

func (s *Service) deleteJobFromQueue(ctx context.Context, queueName, id string) error {
	pipe := s.rdb.TxPipeline()
	pipe.LRem(ctx, s.key(queueName, "wait"), 0, id)
	pipe.LRem(ctx, s.key(queueName, "active"), 0, id)
	pipe.ZRem(ctx, s.key(queueName, "delayed"), id)
	pipe.ZRem(ctx, s.key(queueName, "completed"), id)
	pipe.ZRem(ctx, s.key(queueName, "failed"), id)
	pipe.Del(ctx, s.key(queueName, id))
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete job %s from queue %s: %w", id, queueName, err)
	}
	return nil
}

func (s *Service) key(queueName, suffix string) string {
	return fmt.Sprintf("%s:%s:%s", s.prefix, queueName, suffix)
}
