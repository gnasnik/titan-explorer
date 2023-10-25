package dao

import (
	"context"
	"fmt"
	"github.com/gnasnik/titan-explorer/core/generated/model"
)

var tableNameAsset = "assets"
var tableNameProject = "projects"

func AddAssets(ctx context.Context, assets []*model.Asset) error {
	_, err := DB.NamedExecContext(ctx, fmt.Sprintf(
		`INSERT INTO %s ( node_id, event, cid, hash, total_size, end_time, expiration, user_id, type, name, created_at, updated_at)
			VALUES ( :node_id, :event, :cid, :hash, :total_size, :end_time, :expiration, :user_id, :type, :name, :created_at, :updated_at) 
			ON DUPLICATE KEY UPDATE  event = VALUES(event), end_time = VALUES(end_time), expiration = VALUES(expiration);`, tableNameAsset,
	), assets)
	return err
}

func UpdateAssetPath(ctx context.Context, cid string, path string) error {
	_, err := DB.ExecContext(ctx, fmt.Sprintf(
		`UPDATE %s SET path = ? where cid = ?`, tableNameAsset), path, cid)
	return err
}

func UpdateAssetEvent(ctx context.Context, cid string, event int) error {
	_, err := DB.ExecContext(ctx, fmt.Sprintf(
		`UPDATE %s SET event = ? where cid = ?`, tableNameAsset), event, cid)
	return err
}

func GetLatestAsset(ctx context.Context) (*model.Asset, error) {
	var asset model.Asset
	err := DB.GetContext(ctx, &asset, fmt.Sprintf(
		`SELECT * from %s ORDER BY created_at DESC LIMIT 1`, tableNameAsset))
	if err != nil {
		return nil, err
	}
	return &asset, err
}

func GetAssetsByEmptyPath(ctx context.Context) ([]*model.Asset, int64, error) {
	var out []*model.Asset
	err := DB.SelectContext(ctx, &out, fmt.Sprintf(
		`SELECT * FROM %s WHERE event = 1 AND path = ''`, tableNameAsset,
	))

	if err != nil {
		return nil, 0, err
	}

	var total int64
	err = DB.GetContext(ctx, &total, fmt.Sprintf(
		`SELECT count(*) FROM %s WHERE event = 1 AND path = ''`, tableNameAsset))
	if err != nil {
		return nil, 0, err
	}

	return out, total, err
}

func GetAssetByCID(ctx context.Context, cid string) (*model.Asset, error) {
	var asset model.Asset
	err := DB.GetContext(ctx, &asset, fmt.Sprintf(
		`SELECT * from %s WHERE cid = ?`, tableNameAsset), cid)
	if err != nil {
		return nil, err
	}
	return &asset, err
}

func CountAssets(ctx context.Context) ([]*model.StorageStats, error) {
	queryStatement := fmt.Sprintf(`select a.project_id, s.name, sum(a.total_size) as total_size, now() as time, count(DISTINCT a.user_id) as user_count ,
       count(DISTINCT f.provider) as provider_count, max(f.end_time) as expiration  from %s a inner join %s s on a.project_id = s.id 
       left join %s f on a.path = f.path where a.path <> '' group by a.project_id;`, tableNameAsset, tableNameProject, tableNameFilStorage)

	var out []*model.StorageStats
	if err := DB.SelectContext(ctx, &out, queryStatement); err != nil {
		return nil, err
	}

	return out, nil
}
