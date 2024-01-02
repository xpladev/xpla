package e2e

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
)

const BACK_OFF_MAX_DELAY = 3 * time.Second

type ServiceDesc struct {
	Destination     string
	DestinationPort int
	BackoffMaxDelay time.Duration
	NoTLS           bool

	ServiceConn *grpc.ClientConn
}

func NewServiceDesc(dest string, port int, backoffDelayBySec int, noTls bool) *ServiceDesc {
	serv := &ServiceDesc{
		Destination:     dest,
		DestinationPort: port,
		BackoffMaxDelay: time.Second * time.Duration(backoffDelayBySec),
		NoTLS:           noTls,
	}

	serv.GetConnectionWithContext(context.Background())

	return serv
}

func (desc *ServiceDesc) GetServiceDesc() *ServiceDesc {
	return desc
}

// GetConnectionWithContext returns gRPC client connection with context
func (desc *ServiceDesc) GetConnectionWithContext(ctx context.Context, opts ...grpc.DialOption) *grpc.ClientConn {
	if desc.ServiceConn != nil {
		return desc.ServiceConn
	}

	dest := desc.Destination
	// Assemble port
	dest = strings.Join([]string{dest, strconv.Itoa(desc.DestinationPort)}, ":")

	options := opts
	if !desc.NoTLS {
		// FIXME: it is very slow procedure, we should have cert in local filesystem or cache it
		conn, err := tls.Dial("tcp", dest, &tls.Config{
			InsecureSkipVerify: true,
		})

		if err != nil {
			logger.Fatalln("cannot dial to TLS enabled server:", err)
			return nil
		}

		certs := conn.ConnectionState().PeerCertificates
		err = conn.Close()
		if err != nil {
			logger.Fatalln("cannot close TLS connection:", err)
		}

		pool := x509.NewCertPool()
		pool.AddCert(certs[0])

		clientCert := credentials.NewClientTLSFromCert(pool, "")
		options = append(options, grpc.WithTransportCredentials(clientCert))
	} else {
		options = append(options, grpc.WithInsecure())
	}

	backoff := backoff.DefaultConfig
	// We apply backoff max delay (3 seconds by default)
	backoff.MaxDelay = func() time.Duration {
		delay := desc.BackoffMaxDelay
		if delay != 0 {
			return delay
		}
		return BACK_OFF_MAX_DELAY
	}()

	options = append(options, grpc.WithConnectParams(grpc.ConnectParams{
		Backoff: backoff,
	}))

	logger.Println("dial ", dest)
	conn, err := grpc.DialContext(ctx, dest, options...)
	if err != nil {
		// It is very likely unreachable code under non-blocking dialing
		logger.Panic("cannot connect to gRPC server:", err)
	}
	desc.ServiceConn = conn

	// State change logger
	go func() {
		isReady := false

		for {
			s := conn.GetState()
			if s == connectivity.Shutdown {
				logger.Println("connection state: ", s)
				break
			} else if isReady && s == connectivity.TransientFailure {
				logger.Println("connection state : ", s)
				isReady = false
			}

			if !conn.WaitForStateChange(ctx, s) {
				// Logging last state just after ctx expired
				// Even this can miss last "shutdown" state. very unlikely.
				last := conn.GetState()
				if s != last {
					logger.Println("connection state: ", last)
				}
				break
			}
		}
	}()

	return desc.ServiceConn
}

// CloseConnection closes existing connection
func (desc *ServiceDesc) CloseConnection() error {
	if desc.ServiceConn == nil {
		// Very unlikely
		logger.Fatalln("connection is already closed or not connected yet")
		return nil
	}

	err := desc.ServiceConn.Close()
	if err != nil {
		return err
	}

	desc.ServiceConn = nil
	return nil
}
