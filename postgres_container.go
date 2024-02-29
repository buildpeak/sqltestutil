package sqltestutil

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	waitInterval = 100 * time.Millisecond
	waitTimeout  = 10 * time.Second
)

// PostgresContainer is a Docker container running Postgres. It can be used to
// cheaply start a throwaway Postgres instance for testing.
type PostgresContainer struct {
	id       string
	password string
	port     string
	connStr  string
}

// StartPostgresContainer starts a new Postgres Docker container. The version
// parameter is the tagged version of Postgres image to use, e.g. to use
// postgres:12 pass "12". Creation involes a few steps:
//
// 1. Pull the image if it isn't already cached locally
// 2. Start the container
// 3. Wait for Postgres to be healthy
//
// Once created the container will be immediately usable. It should be stopped
// with the Shutdown method. The container will bind to a randomly available
// host port, and random password. The SQL connection string can be obtained
// with the ConnectionString method.
//
// Container startup and shutdown together can take a few seconds (longer when
// the image has not yet been pulled.) This is generally too slow to initiate in
// each unit test so it's advisable to do setup and teardown once for a whole
// suite of tests. TestMain is one way to do this, however because of Golang
// issue 37206 [1], panics in tests will immediately exit the process without
// giving you the opportunity to Shutdown, which results in orphaned containers
// lying around.
//
// Another approach is to write a single test that starts and stops the
// container and then run sub-tests within there. The testify [2] suite
// package provides a good way to structure these kinds of tests:
//
//	type ExampleTestSuite struct {
//	    suite.Suite
//	}
//
//	func (s *ExampleTestSuite) TestExample() {
//	    // test something
//	}
//
//	func TestExampleTestSuite(t *testing.T) {
//	    pg, _ := sqltestutil.StartPostgresContainer(context.Background(), "12")
//	    defer pg.Shutdown(ctx)
//	    suite.Run(t, &ExampleTestSuite{})
//	}
//
// [1]: https://github.com/golang/go/issues/37206
// [2]: https://github.com/stretchr/testify
func StartPostgresContainer(ctx context.Context, version string) (*PostgresContainer, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}
	defer cli.Close()
	image := "postgres:" + version
	_, _, err = cli.ImageInspectWithRaw(ctx, image)
	if err != nil {
		_, notFound := err.(interface {
			NotFound()
		})
		if !notFound {
			return nil, err
		}
		pullReader, err := cli.ImagePull(ctx, image, types.ImagePullOptions{})
		if err != nil {
			return nil, err
		}
		_, err = io.Copy(io.Discard, pullReader)
		pullReader.Close()
		if err != nil {
			return nil, err
		}
	}

	password, err := randomPassword()
	if err != nil {
		return nil, err
	}
	port, err := randomPort()
	if err != nil {
		return nil, err
	}
	createResp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: image,
		Env: []string{
			"POSTGRES_DB=pgtest",
			"POSTGRES_PASSWORD=" + password,
			"POSTGRES_USER=pgtest",
		},
		Healthcheck: &container.HealthConfig{
			Test:     []string{"CMD-SHELL", "pg_isready -U pgtest"},
			Interval: time.Second,
			Timeout:  time.Second,
			Retries:  10,
		},
	}, &container.HostConfig{
		PortBindings: nat.PortMap{
			"5432/tcp": []nat.PortBinding{
				{HostPort: port},
			},
		},
	}, nil, nil, "")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			removeErr := cli.ContainerRemove(ctx, createResp.ID, types.ContainerRemoveOptions{})
			if removeErr != nil {
				fmt.Println("error removing container:", removeErr)
				return
			}
		}
	}()
	err = cli.ContainerStart(ctx, createResp.ID, types.ContainerStartOptions{})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			stopErr := cli.ContainerStop(ctx, createResp.ID, nil)
			if stopErr != nil {
				fmt.Println("error stopping container:", stopErr)
				return
			}
		}
	}()

	ctx, cancel := context.WithTimeout(ctx, waitTimeout)
	defer cancel()

HealthCheck:
	for {
		// Check if the context has been cancelled before each health check
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		inspect, err := cli.ContainerInspect(ctx, createResp.ID)
		if err != nil {
			return nil, err
		}
		status := inspect.State.Health.Status
		switch status {
		case "unhealthy":
			return nil, errors.New("container unhealthy")
		case "healthy":
			break HealthCheck
		default:
			time.Sleep(waitInterval)
		}
	}

	connStr := fmt.Sprintf("postgres://pgtest:%s@127.0.0.1:%s/pgtest", password, port)
	err = waitUntilConnectable(ctx, connStr)
	if err != nil {
		return nil, err
	}

	return &PostgresContainer{
		id:       createResp.ID,
		password: password,
		port:     port,
		connStr:  connStr,
	}, nil
}

// ConnectionString returns a connection URL string that can be used to connect
// to the running Postgres container.
func (c *PostgresContainer) ConnectionString() string {
	return c.connStr
}

// ID returns the Docker container ID of the running Postgres container.
func (c *PostgresContainer) ID() string {
	return c.id
}

// Shutdown cleans up the Postgres container by stopping and removing it. This
// should be called each time a PostgresContainer is created to avoid orphaned
// containers.
func (c *PostgresContainer) Shutdown(ctx context.Context) error {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return err
	}
	defer cli.Close()
	err = cli.ContainerStop(ctx, c.id, nil)
	if err != nil {
		return err
	}
	err = cli.ContainerRemove(ctx, c.id, types.ContainerRemoveOptions{})
	if err != nil {
		return err
	}
	return nil
}

func waitUntilConnectable(ctx context.Context, connStr string) error {
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return err
	}
	defer db.Close()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := db.PingContext(ctx)
		if err == nil {
			return nil
		}
		time.Sleep(waitInterval)
	}
}

var passwordLetters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randomPassword() (string, error) {
	const passwordLength = 32
	b := make([]rune, passwordLength)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(passwordLetters))))
		if err != nil {
			return "", err
		}
		b[i] = passwordLetters[n.Int64()]
	}
	return string(b), nil
}

func randomPort() (string, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer l.Close()
	_, port, err := net.SplitHostPort(l.Addr().String())
	return port, err
}
