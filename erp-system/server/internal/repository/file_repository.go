// 有问题欢迎联系 hot_kun@hotmail.com
package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"practical-training/erp-system/internal/domain"
)

type FileRepository struct {
	db *sql.DB
}

func NewFileRepository(db *sql.DB) *FileRepository {
	return &FileRepository{db: db}
}

const fileColumns = `id, created_at, group_name, original_name, storage_path,
	size, content_type, uploader_id, uploader_name`

func scanFile(s interface{ Scan(...any) error }) (domain.File, error) {
	var f domain.File
	err := s.Scan(&f.ID, &f.CreatedAt, &f.GroupName, &f.OriginalName, &f.StoragePath,
		&f.Size, &f.ContentType, &f.UploaderID, &f.UploaderName)
	return f, err
}

func (r *FileRepository) Create(ctx context.Context, f domain.File) (domain.File, error) {
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO sys_files (group_name, original_name, storage_path, size, content_type, uploader_id, uploader_name)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		f.GroupName, f.OriginalName, f.StoragePath, f.Size, f.ContentType, f.UploaderID, f.UploaderName,
	)
	if err != nil {
		return domain.File{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.File{}, err
	}
	f.ID = id
	if err := r.db.QueryRowContext(ctx,
		`SELECT created_at FROM sys_files WHERE id = ?`, id).Scan(&f.CreatedAt); err != nil {
		return domain.File{}, err
	}
	return f, nil
}

// CreateWithinQuota 在单事务里完成「总量 + 本次是否超配额」的判断并登记。
//
// 不能用「先 Usage 读一次、再 Create」：两个并发上传会读到同一个 used，
// 各自都算得下，一起写进去就超了配额。
//
// SQLite 版把判断塞进 INSERT ... SELECT 的 WHERE 里、靠单写者模型串行化；
// MySQL 没有这个前提，改为事务内 SELECT ... FOR UPDATE 锁住现有文件行：
// 在 REPEATABLE READ 下，全表扫描会带上间隙锁，空表时锁 supremum，
// 因此并发插入会被挡在锁外，读到的是对方提交后的总量。
//
// quota <= 0 表示不限总量，此时不参与条件判断。
// 超配额时返回 domain.ErrQuotaExceeded。
func (r *FileRepository) CreateWithinQuota(ctx context.Context, f domain.File, quota int64) (domain.File, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.File{}, err
	}
	defer tx.Rollback() //nolint:errcheck // Commit 成功后这里是空操作

	if quota > 0 {
		var used int64
		if err := tx.QueryRowContext(ctx,
			`SELECT COALESCE(SUM(size), 0) FROM sys_files FOR UPDATE`).Scan(&used); err != nil {
			return domain.File{}, err
		}
		if used+f.Size > quota {
			return domain.File{}, domain.ErrQuotaExceeded
		}
	}

	res, err := tx.ExecContext(ctx, `
		INSERT INTO sys_files (group_name, original_name, storage_path, size, content_type, uploader_id, uploader_name)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		f.GroupName, f.OriginalName, f.StoragePath, f.Size, f.ContentType, f.UploaderID, f.UploaderName,
	)
	if err != nil {
		return domain.File{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.File{}, err
	}
	f.ID = id
	if err := tx.QueryRowContext(ctx,
		`SELECT created_at FROM sys_files WHERE id = ?`, id).Scan(&f.CreatedAt); err != nil {
		return domain.File{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.File{}, err
	}
	return f, nil
}

func (r *FileRepository) GetByID(ctx context.Context, id int64) (domain.File, error) {
	f, err := scanFile(r.db.QueryRowContext(ctx,
		`SELECT `+fileColumns+` FROM sys_files WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.File{}, domain.ErrNotFound
	}
	return f, err
}

// FileFilter 是文件列表的筛选条件。
type FileFilter struct {
	OriginalName string // 模糊
	GroupName    string // 精确
}

func fileWhere(f FileFilter) (string, []any) {
	var b whereBuilder
	b.like("original_name", f.OriginalName)
	b.eq("group_name", f.GroupName)
	return b.clause(), b.args
}

func (r *FileRepository) Count(ctx context.Context, f FileFilter) (int64, error) {
	where, args := fileWhere(f)

	var n int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sys_files`+where, args...).Scan(&n)
	return n, err
}

func (r *FileRepository) List(ctx context.Context, f FileFilter, limit, offset int) ([]domain.File, error) {
	where, args := fileWhere(f)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx,
		`SELECT `+fileColumns+` FROM sys_files`+where+` ORDER BY id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]domain.File, 0, limit)
	for rows.Next() {
		item, err := scanFile(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

// Usage 返回总占用字节数与文件个数。
func (r *FileRepository) Usage(ctx context.Context) (int64, int64, error) {
	var used, count sql.NullInt64
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(size), 0), COUNT(*) FROM sys_files`).Scan(&used, &count)
	if err != nil {
		return 0, 0, err
	}
	return used.Int64, count.Int64, nil
}

func (r *FileRepository) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM sys_files WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return requireAffected(res)
}

// AllStoragePaths 返回全部磁盘相对路径，供孤儿清理比对。
//
// 一次性载入内存：这是**删除孤儿文件**用的，只有清理任务会调，
// 而且路径很短。按十万个文件算也就几 MB，比逐条查库再比对简单得多。
// ExistsStoragePath 判断某个磁盘路径是否已有记录。
//
// 给孤儿清理的「删除前最后一次确认」用：从拿到清单到轮到某个文件，
// 中间隔了整个目录遍历的时间，足够一个并发上传把自己的登记写完。
func (r *FileRepository) ExistsStoragePath(ctx context.Context, storagePath string) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM sys_files WHERE storage_path = ?)`, storagePath).Scan(&ok)
	return ok, err
}

// List 返回磁盘上的全部文件相对路径，供孤儿清理比对。
//
// 先拿到磁盘清单再查库，是清理流程里唯一能缩窄竞态窗口的顺序：
// 反过来（先查库再遍历磁盘）会把「查库之后、遍历之前」上传的文件当成孤儿。
// 这个方向下，最坏情况是刚被删掉的文件本轮没被收，下轮再收——无害。
func (r *FileRepository) AllStoragePaths(ctx context.Context) (map[string]struct{}, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT storage_path FROM sys_files`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]struct{}{}
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		out[p] = struct{}{}
	}
	return out, rows.Err()
}

// StoragePathsCreatedBefore 返回创建时间早于阈值的文件路径集合。
//
// 只让「有年代」的文件可能被当孤儿清理：刚上传、元数据还没写完的文件
// 不会落入这个集合，于是不会被并发跑的清理任务误删。
func (r *FileRepository) StoragePathsCreatedBefore(ctx context.Context, before time.Time) (map[string]struct{}, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT storage_path FROM sys_files WHERE created_at < ?`, toDBTime(before))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]struct{}{}
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		out[p] = struct{}{}
	}
	return out, rows.Err()
}
