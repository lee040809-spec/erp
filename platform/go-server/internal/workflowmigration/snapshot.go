// Shared read-only row/column snapshots for both migration verification tools.
package workflowmigration

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
)

type TableCheck struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Rows    int64    `json:"rows"`
	Hash    string   `json:"sha256"`
}

func Snapshot(ctx context.Context, db *sql.DB, reference []TableCheck) ([]TableCheck, error) {
	if reference == nil {
		rows, err := db.QueryContext(ctx, "SELECT table_name FROM information_schema.tables WHERE table_schema=DATABASE() AND table_type='BASE TABLE' ORDER BY table_name")
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var name string
			if err = rows.Scan(&name); err != nil {
				rows.Close()
				return nil, err
			}
			reference = append(reference, TableCheck{Name: name})
		}
		if err = rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
	}
	result := []TableCheck{}
	for _, ref := range reference {
		cols := ref.Columns
		if len(cols) == 0 {
			rows, err := db.QueryContext(ctx, "SELECT column_name FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name=? ORDER BY ordinal_position", ref.Name)
			if err != nil {
				return nil, err
			}
			for rows.Next() {
				var s string
				if err = rows.Scan(&s); err != nil {
					rows.Close()
					return nil, err
				}
				cols = append(cols, s)
			}
			if err = rows.Err(); err != nil {
				rows.Close()
				return nil, err
			}
			rows.Close()
		}
		quoted := []string{}
		for _, c := range cols {
			quoted = append(quoted, "`"+strings.ReplaceAll(c, "`", "``")+"`")
		}
		// Existing ERP tables have primary IDs/company keys. Order all selected columns for deterministic snapshots.
		q := "SELECT " + strings.Join(quoted, ",") + " FROM `" + ref.Name + "` ORDER BY " + strings.Join(quoted, ",")
		rows, err := db.QueryContext(ctx, q)
		if err != nil {
			return nil, err
		}
		hash := sha256.New()
		var count int64
		for rows.Next() {
			vals := make([]sql.RawBytes, len(cols))
			ptr := make([]interface{}, len(cols))
			for i := range vals {
				ptr[i] = &vals[i]
			}
			if err = rows.Scan(ptr...); err != nil {
				rows.Close()
				return nil, err
			}
			for _, v := range vals {
				if v == nil {
					fmt.Fprint(hash, "N;")
				} else {
					fmt.Fprintf(hash, "%d:", len(v))
					hash.Write(v)
					hash.Write([]byte{';'})
				}
			}
			hash.Write([]byte{'\n'})
			count++
		}
		if err = rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
		result = append(result, TableCheck{Name: ref.Name, Columns: cols, Rows: count, Hash: hex.EncodeToString(hash.Sum(nil))})
	}
	return result, nil
}
