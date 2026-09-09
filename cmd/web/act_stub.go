package main

// Throwaway prototype for #1791. Lands nothing. A stub recorder, shaped as #1788 ruled it: one
// method name, two receivers — pool-bound by default, tx-bound on the restore path. It is a
// concrete value type so the zero server the existing tests build keeps working.

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type stubRecorder struct{}

func (stubRecorder) Record(context.Context, string, string) error { return nil }

func (r stubRecorder) WithTx(pgx.Tx) stubRecorder { return r }
