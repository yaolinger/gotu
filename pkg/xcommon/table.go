package xcommon

import (
	"context"
	"fmt"
	"gotu/pkg/xlog"

	"github.com/liushuochen/gotable"
	"go.uber.org/zap"
)

func printTable(ctx context.Context, keys []string, values [][]string) error {
	table, err := gotable.CreateSafeTable(keys...)
	if err != nil {
		return err
	}
	for _, vs := range values {
		if err := table.AddRow(vs); err != nil {
			return err
		}
	}
	fmt.Println(table)
	return nil
}

func PrintTable(ctx context.Context, keys []string, values [][]string) {
	if err := printTable(ctx, keys, values); err != nil {
		xlog.Get(ctx).Warn("Print table failed.", zap.Any("err", err))
	}
}
