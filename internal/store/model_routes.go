package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"local-ai-gateway/internal/model"
)

func (s *Store) ListModelRoutes(ctx context.Context) ([]model.ModelRoute, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id,name,enabled,created_at,updated_at
		FROM model_routes ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	routes := make([]model.ModelRoute, 0)
	index := make(map[string]int)
	for rows.Next() {
		var item model.ModelRoute
		var enabled int
		var created, updated string
		if err := rows.Scan(&item.ID, &item.Name, &enabled, &created, &updated); err != nil {
			_ = rows.Close()
			return nil, err
		}
		item.Enabled = enabled != 0
		item.CreatedAt = parseTime(created)
		item.UpdatedAt = parseTime(updated)
		item.Models = []model.ModelRouteModel{}
		index[item.ID] = len(routes)
		routes = append(routes, item)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	modelRows, err := s.db.QueryContext(ctx, `
		SELECT route_id,name,priority,enabled
		FROM model_route_models ORDER BY route_id,priority DESC,name
	`)
	if err != nil {
		return nil, err
	}
	modelIndexes := make(map[string]map[string]int)
	for modelRows.Next() {
		var routeID string
		var item model.ModelRouteModel
		var enabled int
		if err := modelRows.Scan(&routeID, &item.Name, &item.Priority, &enabled); err != nil {
			_ = modelRows.Close()
			return nil, err
		}
		routeIndex, ok := index[routeID]
		if !ok {
			continue
		}
		item.Enabled = enabled != 0
		item.Targets = []model.ModelRouteTarget{}
		if modelIndexes[routeID] == nil {
			modelIndexes[routeID] = make(map[string]int)
		}
		modelIndexes[routeID][item.Name] = len(routes[routeIndex].Models)
		routes[routeIndex].Models = append(routes[routeIndex].Models, item)
	}
	if err := modelRows.Close(); err != nil {
		return nil, err
	}
	if err := modelRows.Err(); err != nil {
		return nil, err
	}

	targetRows, err := s.db.QueryContext(ctx, `
		SELECT t.route_id,t.route_model,t.provider_id,t.upstream_model,t.enabled
		FROM model_route_targets t
		JOIN providers p ON p.id=t.provider_id
		ORDER BY t.route_id,t.route_model,p.priority DESC,t.provider_id,t.upstream_model
	`)
	if err != nil {
		return nil, err
	}
	defer targetRows.Close()
	for targetRows.Next() {
		var routeID, routeModel string
		var item model.ModelRouteTarget
		var enabled int
		if err := targetRows.Scan(&routeID, &routeModel, &item.ProviderID, &item.UpstreamModel, &enabled); err != nil {
			return nil, err
		}
		routeIndex, routeOK := index[routeID]
		modelIndex, modelOK := modelIndexes[routeID][routeModel]
		if !routeOK || !modelOK {
			continue
		}
		item.Enabled = enabled != 0
		routes[routeIndex].Models[modelIndex].Targets = append(routes[routeIndex].Models[modelIndex].Targets, item)
	}
	return routes, targetRows.Err()
}

func (s *Store) GetModelRoute(ctx context.Context, id string) (*model.ModelRoute, error) {
	routes, err := s.ListModelRoutes(ctx)
	if err != nil {
		return nil, err
	}
	for i := range routes {
		if routes[i].ID == id {
			return &routes[i], nil
		}
	}
	return nil, sql.ErrNoRows
}

func (s *Store) UpsertModelRoute(ctx context.Context, route model.ModelRoute) (model.ModelRoute, error) {
	route.ID = strings.TrimSpace(route.ID)
	route.Name = strings.TrimSpace(route.Name)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return route, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO model_routes (id,name,enabled,updated_at)
		VALUES (?,?,?,CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET name=excluded.name,enabled=excluded.enabled,updated_at=CURRENT_TIMESTAMP
	`, route.ID, route.Name, boolInt(route.Enabled)); err != nil {
		return route, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM model_route_models WHERE route_id=?`, route.ID); err != nil {
		return route, err
	}
	for _, routeModel := range route.Models {
		routeModel.Name = strings.TrimSpace(routeModel.Name)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO model_route_models (route_id,name,priority,enabled) VALUES (?,?,?,?)
		`, route.ID, routeModel.Name, routeModel.Priority, boolInt(routeModel.Enabled)); err != nil {
			return route, err
		}
		for _, target := range routeModel.Targets {
			target.ProviderID = strings.TrimSpace(target.ProviderID)
			target.UpstreamModel = strings.TrimSpace(target.UpstreamModel)
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO model_route_targets (route_id,route_model,provider_id,upstream_model,enabled)
				VALUES (?,?,?,?,?)
			`, route.ID, routeModel.Name, target.ProviderID, target.UpstreamModel, boolInt(target.Enabled)); err != nil {
				return route, err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return route, err
	}
	saved, err := s.GetModelRoute(ctx, route.ID)
	if err != nil {
		return route, err
	}
	return *saved, nil
}

func (s *Store) DeleteModelRoute(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM model_routes WHERE id=?`, strings.TrimSpace(id))
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func IsModelRouteNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
