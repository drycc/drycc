package main

import (
	"net/http"
	"os"
	"time"

	"github.com/drycc/drycc/controller/client"
	"github.com/drycc/drycc/controller/schema"
	"github.com/drycc/drycc/controller/worker/app_deletion"
	"github.com/drycc/drycc/controller/worker/app_garbage_collection"
	"github.com/drycc/drycc/controller/worker/deployment"
	"github.com/drycc/drycc/controller/worker/domain_migration"
	"github.com/drycc/drycc/controller/worker/release_cleanup"
	"github.com/drycc/drycc/discoverd/client"
	"github.com/drycc/drycc/pkg/postgres"
	"github.com/drycc/drycc/pkg/shutdown"
	"github.com/drycc/drycc/pkg/status"
	"github.com/drycc/que-go"
	"github.com/inconshreveable/log15"
)

const workerCount = 10

var logger = log15.New("app", "worker")

func main() {
	log := logger.New("fn", "main")

	log.Info("creating controller client")
	client, err := controller.NewClient("", os.Getenv("AUTH_KEY"))
	if err != nil {
		log.Error("error creating controller client", "err", err)
		shutdown.Fatal(err)
	}

	log.Info("connecting to postgres")
	db := postgres.Wait(nil, schema.PrepareStatements)

	shutdown.BeforeExit(func() { db.Close() })

	go func() {
		status.AddHandler(func() status.Status {
			_, err := db.ConnPool.Exec("ping")
			if err != nil {
				return status.Unhealthy
			}
			return status.Healthy
		})
		addr := ":" + os.Getenv("PORT")
		hb, err := discoverd.AddServiceAndRegister("controller-worker", addr)
		if err != nil {
			shutdown.Fatal(err)
		}
		shutdown.BeforeExit(func() { hb.Close() })
		shutdown.Fatal(http.ListenAndServe(addr, nil))
	}()

	workers := que.NewWorkerPool(
		que.NewClient(db.ConnPool),
		que.WorkMap{
			"deployment":             deployment.JobHandler(db, client, logger),
			"app_deletion":           app_deletion.JobHandler(db, client, logger),
			"domain_migration":       domain_migration.JobHandler(db, client, logger),
			"release_cleanup":        release_cleanup.JobHandler(db, client, logger),
			"app_garbage_collection": app_garbage_collection.JobHandler(db, client, logger),
		},
		workerCount,
	)
	workers.Interval = 5 * time.Second

	log.Info("starting workers", "count", workerCount, "interval", workers.Interval)
	workers.Start()
	shutdown.BeforeExit(func() { workers.Shutdown() })

	select {} // block and keep running
}
