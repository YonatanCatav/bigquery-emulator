package server_test

import (
	"context"
	"testing"

	"cloud.google.com/go/bigquery"
	"github.com/goccy/bigquery-emulator/server"
	"github.com/goccy/bigquery-emulator/types"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func TestHashAsIntRepro(t *testing.T) {
	ctx := context.Background()

	const (
		projectName = "test"
		datasetName = "test"
	)

	bqServer, err := server.New(server.TempStorage)
	if err != nil {
		t.Fatal(err)
	}
	if err := bqServer.Load(
		server.StructSource(
			types.NewProject(
				projectName,
				types.NewDataset(
					datasetName,
					types.NewTable(
						"ad_objects_default_view",
						[]*types.Column{
							types.NewColumn("guid", types.STRING),
						},
						types.Data{
							{
								"guid": "test-guid",
							},
						},
					),
				),
			),
		),
	); err != nil {
		t.Fatal(err)
	}
	testServer := bqServer.TestServer()
	defer func() {
		testServer.Close()
		bqServer.Close()
	}()

	client, err := bigquery.NewClient(
		ctx,
		projectName,
		option.WithEndpoint(testServer.URL),
		option.WithoutAuthentication(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	// 1. Create the functions
	q1 := client.Query("CREATE OR REPLACE FUNCTION test.hash_as_int(arr any type) returns INT64 as (test.md5ToInt(array_to_string(arr, '*', '*')))")
	if _, err := q1.Read(ctx); err != nil {
		t.Fatalf("failed to create hash_as_int function: %v", err)
	}

	// 2. Run the failing query
	q2 := client.Query("SELECT test.hash_as_int([lower(CAST(concat('guid_', `test.test.ad_objects_default_view`.`guid`) AS STRING))]) AS `_id` FROM `test.test.ad_objects_default_view` WHERE `test.test.ad_objects_default_view`.`guid` IS NOT NULL")
	it, err := q2.Read(ctx)
	if err != nil {
		t.Fatalf("failed to execute query: %v", err)
	}

	for {
		var row []bigquery.Value
		if err := it.Next(&row); err != nil {
			if err == iterator.Done {
				break
			}
			t.Fatal(err)
		}
		t.Log("row = ", row)
	}
}
