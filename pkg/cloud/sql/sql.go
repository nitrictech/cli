package sql

import (
	"context"

	sqlpb "github.com/nitrictech/nitric/core/pkg/proto/sql/v1"
)

type LocalSqlServer struct {
	sqlpb.UnimplementedSqlServer
}

var _ sqlpb.SqlServer = (*LocalSqlServer)(nil)

func (*LocalSqlServer) ConnectionString(context.Context, *sqlpb.SqlConnectionStringRequest) (*sqlpb.SqlConnectionStringResponse, error) {
	// We can lazily create a new database instance here and return the connection information directly
	return &sqlpb.SqlConnectionStringResponse{
		ConnectionString: "",
	}, nil
}
