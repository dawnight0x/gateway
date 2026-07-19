package store

import (
	"context"
	"database/sql"
	"strings"

	"local-ai-gateway/internal/model"
)

type LogQuery struct {
	Limit      int
	Offset     int
	Status     *int
	ProviderID string
	KeyID      string
	Model      string
	ErrorType  string
	Search     string
}

type LogPage struct {
	Items  []model.RequestLog `json:"items"`
	Total  int                `json:"total"`
	Limit  int                `json:"limit"`
	Offset int                `json:"offset"`
}

func (s *Store) QueryLogs(ctx context.Context, query LogQuery) (LogPage, error) {
	if err := s.flushRequestLogs(ctx); err != nil {
		return LogPage{}, err
	}
	if query.Limit < 1 {
		query.Limit = 100
	}
	if query.Limit > 10000 {
		query.Limit = 10000
	}
	if query.Offset < 0 {
		query.Offset = 0
	}

	where, args := logWhere(query)
	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM request_logs`+where, args...).Scan(&total); err != nil {
		return LogPage{}, err
	}
	pageArgs := append(append([]any(nil), args...), query.Limit, query.Offset)
	rows, err := s.db.QueryContext(ctx, `SELECT id,request_id,inbound_protocol,provider_id,key_id,model,route_id,upstream_model,attempts,status,latency_ms,prompt_tokens,completion_tokens,total_tokens,error_type,created_at FROM request_logs`+where+` ORDER BY id DESC LIMIT ? OFFSET ?`, pageArgs...)
	if err != nil {
		return LogPage{}, err
	}
	defer rows.Close()
	items, err := scanRequestLogRows(rows)
	if err != nil {
		return LogPage{}, err
	}
	return LogPage{Items: items, Total: total, Limit: query.Limit, Offset: query.Offset}, nil
}

func (s *Store) DeleteRequestLogs(ctx context.Context) (int64, error) {
	if err := s.flushRequestLogs(ctx); err != nil {
		return 0, err
	}
	result, err := s.db.ExecContext(ctx, `DELETE FROM request_logs`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func logWhere(query LogQuery) (string, []any) {
	conditions := make([]string, 0, 6)
	args := make([]any, 0, 8)
	if query.Status != nil {
		conditions = append(conditions, `status=?`)
		args = append(args, *query.Status)
	}
	for column, value := range map[string]string{
		"provider_id": query.ProviderID,
		"key_id":      query.KeyID,
		"model":       query.Model,
		"error_type":  query.ErrorType,
	} {
		if value = strings.TrimSpace(value); value != "" {
			conditions = append(conditions, column+`=?`)
			args = append(args, value)
		}
	}
	if search := strings.TrimSpace(query.Search); search != "" {
		like := "%" + search + "%"
		conditions = append(conditions, `(request_id LIKE ? OR model LIKE ? OR route_id LIKE ? OR upstream_model LIKE ? OR provider_id LIKE ? OR key_id LIKE ? OR error_type LIKE ?)`)
		args = append(args, like, like, like, like, like, like, like)
	}
	if len(conditions) == 0 {
		return "", args
	}
	return ` WHERE ` + strings.Join(conditions, ` AND `), args
}

func scanRequestLogRows(rows *sql.Rows) ([]model.RequestLog, error) {
	out := make([]model.RequestLog, 0)
	for rows.Next() {
		var item model.RequestLog
		var providerID, keyID, modelID, routeID, upstreamModel sql.NullString
		var prompt, completion, total sql.NullInt64
		var created string
		if err := rows.Scan(&item.ID, &item.RequestID, &item.InboundProtocol, &providerID, &keyID, &modelID, &routeID, &upstreamModel, &item.Attempts, &item.Status, &item.LatencyMS, &prompt, &completion, &total, &item.ErrorType, &created); err != nil {
			return nil, err
		}
		item.ProviderID = providerID.String
		item.KeyID = keyID.String
		item.Model = modelID.String
		item.RouteID = routeID.String
		item.UpstreamModel = upstreamModel.String
		item.PromptTokens = intPtr(prompt)
		item.CompletionTokens = intPtr(completion)
		item.TotalTokens = intPtr(total)
		item.CreatedAt = parseTime(created)
		out = append(out, item)
	}
	return out, rows.Err()
}
