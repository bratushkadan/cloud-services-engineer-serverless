package main

import (
	// importing the packages ydb-go-sdk
	"context"
	"log"
	"net/http"
	"os"
	"path"

	// "github.com/ydb-platform/ydb-go-sdk-auth-environ" // needed to authenticate using environment variables
	"github.com/ydb-platform/ydb-go-sdk/v3" // needed to work with table service
	"github.com/ydb-platform/ydb-go-sdk/v3/table"
	"github.com/ydb-platform/ydb-go-sdk/v3/table/options"
	"github.com/ydb-platform/ydb-go-sdk/v3/table/result/named"
	"github.com/ydb-platform/ydb-go-sdk/v3/table/types"
	yc "github.com/ydb-platform/ydb-go-yc"
	// needed to work with table service
	// needed to work with table service
	// needed to work with table service
	// needed to work with YDB types and values
	// "github.com/ydb-platform/ydb-go-yc" // to work with YDB in Yandex Cloud
)

var (
	createTable = false
	selectRows  = true
)

// func main() {
// 	if err := run(); err != nil {
// 		log.Fatal(err)
// 	}
// }

func Handler(w http.ResponseWriter, r *http.Request) {
	if err := run(); err != nil {
		log.Fatal(err)
	}

	w.Write([]byte(`{"ok": true}`))
}

func run() error {
	ctx := context.Background()
	// connection string
	dsn := os.Getenv("YDB_ENDPOINT")
	// IAM token
	// token := "t1.9euelZq..."
	// create a connection object called db, it is an entry point for YDB services
	db, err := ydb.Open(ctx, dsn,
		//  yc.WithInternalCA(), // use Yandex Cloud certificates
		//ydb.WithAccessTokenCredentials(token), // authenticate using the token
		//  ydb.WithAnonimousCredentials(), // authenticate anonymously (for example, using docker ydb)
		yc.WithMetadataCredentials(), // authenticate from inside a VM in Yandex Cloud or Yandex Function
	//  yc.WithServiceAccountKeyFileCredentials("~/.ydb/sa.json"), // authenticate in Yandex Cloud using a service account file
	//  environ.WithEnvironCredentials(ctx), // authenticate using environment variables
	)
	if err != nil {
		log.Fatal(err)
	}
	// driver must be closed when done
	defer db.Close(ctx)

	if createTable {
		if err := db.Table().Do(ctx, func(ctx context.Context, s table.Session) error {
			return s.CreateTable(ctx, path.Join(db.Name(), "users"),
				options.WithColumn("id", types.TypeUint64),
				options.WithColumn("name", types.Optional(types.TypeUTF8)),
				options.WithPrimaryKeyColumn("id"),
			)
		}); err != nil {
			return err
		}
	}

	readTx := table.TxControl(table.BeginTx(table.WithOnlineReadOnly()), table.CommitTx())

	if err := db.Table().Do(ctx, func(ctx context.Context, s table.Session) error {
		_, res, err := s.Execute(ctx, readTx, `
        DECLARE $username AS Utf8;
        SELECT
          id,
          name
        FROM
          users
        WHERE
          name = $username;
        `, table.NewQueryParameters(table.ValueParam("$username", types.UTF8Value("Dan"))),
		)
		if err != nil {
			return err
		}
		defer res.Close()
		var (
			id   uint64
			name *string
		)
		for res.NextResultSet(ctx) {
			for res.NextRow() {
				if err := res.ScanNamed(
					named.Required("id", &id),
					named.Optional("name", &name),
				); err != nil {
					return err
				}
				log.Printf(`users row: id="%d", name="%s"`, id, *name)
			}
		}

		return res.Err()
	}); err != nil {
		return err
	}

	return nil
}
