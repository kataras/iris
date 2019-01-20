package httptest

import (
	"crypto/tls"
	"net"
)

// copied from net/http/httptest/internal

// LocalhostCert is a PEM-encoded TLS cert with SAN IPs
// "127.0.0.1" and "[::1]", expiring at Jan 29 16:00:00 2084 GMT.
// generated from src/crypto/tls:
// go run generate_cert.go  --rsa-bits 1024 --host 127.0.0.1,::1,example.com --ca --start-date "Jan 1 00:00:00 1970" --duration=1000000h
// note: these are not the net/http/httptest/internal contents but doesn't matter.
var LocalhostCert = []byte(`-----BEGIN CERTIFICATE-----
MIIDAzCCAeugAwIBAgIJAP0pWSuIYyQCMA0GCSqGSIb3DQEBBQUAMBgxFjAUBgNV
BAMMDWxvY2FsaG9zdDozMzEwHhcNMTYxMjI1MDk1OTI3WhcNMjYxMjIzMDk1OTI3
WjAYMRYwFAYDVQQDDA1sb2NhbGhvc3Q6MzMxMIIBIjANBgkqhkiG9w0BAQEFAAOC
AQ8AMIIBCgKCAQEA5vETjLa+8W856rWXO1xMF/CLss9vn5xZhPXKhgz+D7ogSAXm
mWP53eeBUGC2r26J++CYfVqwOmfJEu9kkGUVi8cGMY9dHeIFPfxD31MYX175jJQe
tu0WeUII7ciNsSUDyBMqsl7yi1IgN7iLONM++1+QfbbmNiEbghRV6icEH6M+bWlz
3YSAMEdpK3mg2gsugfLKMwJkaBKEehUNMySRlIhyLITqt1exYGaggRd1zjqUpqpD
sL2sRVHJ3qHGkSh8nVy8MvG8BXiFdYQJP3mCQDZzruCyMWj5/19KAyu7Cto3Bcvu
PgujnwRtU+itt8WhZUVtU1n7Ivf6lMJTBcc4OQIDAQABo1AwTjAdBgNVHQ4EFgQU
MXrBvbILQmiwjUj19aecF2N+6IkwHwYDVR0jBBgwFoAUMXrBvbILQmiwjUj19aec
F2N+6IkwDAYDVR0TBAUwAwEB/zANBgkqhkiG9w0BAQUFAAOCAQEA4zbFml1t9KXJ
OijAV8gALePR8v04DQwJP+jsRxXw5zzhc8Wqzdd2hjUd07mfRWAvmyywrmhCV6zq
OHznR+aqIqHtm0vV8OpKxLoIQXavfBd6axEXt3859RDM4xJNwIlxs3+LWGPgINud
wjJqjyzSlhJpQpx4YZ5Da+VMiqAp8N1UeaZ5lBvmSDvoGh6HLODSqtPlWMrldRW9
AfsXVxenq81MIMeKW2fSOoPnWZ4Vjf1+dSlbJE/DD4zzcfbyfgY6Ep/RrUltJ3ag
FQbuNTQlgKabe21dSL9zJ2PengVKXl4Trl+4t/Kina9N9Jw535IRCSwinD6a/2Ca
m7DnVXFiVA==
-----END CERTIFICATE-----
`)

// LocalhostKey is the private key for localhostCert.
var LocalhostKey = []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA5vETjLa+8W856rWXO1xMF/CLss9vn5xZhPXKhgz+D7ogSAXm
mWP53eeBUGC2r26J++CYfVqwOmfJEu9kkGUVi8cGMY9dHeIFPfxD31MYX175jJQe
tu0WeUII7ciNsSUDyBMqsl7yi1IgN7iLONM++1+QfbbmNiEbghRV6icEH6M+bWlz
3YSAMEdpK3mg2gsugfLKMwJkaBKEehUNMySRlIhyLITqt1exYGaggRd1zjqUpqpD
sL2sRVHJ3qHGkSh8nVy8MvG8BXiFdYQJP3mCQDZzruCyMWj5/19KAyu7Cto3Bcvu
PgujnwRtU+itt8WhZUVtU1n7Ivf6lMJTBcc4OQIDAQABAoIBAQCTLE0eHpPevtg0
+FaRUMd5diVA5asoF3aBIjZXaU47bY0G+SO02x6wSMmDFK83a4Vpy/7B3Bp0jhF5
DLCUyKaLdmE/EjLwSUq37ty+JHFizd7QtNBCGSN6URfpmSabHpCjX3uVQqblHIhF
mki3BQCdJ5CoXPemxUCHjDgYSZb6JVNIPJExjekc0+4A2MYWMXV6Wr86C7AY3659
KmveZpC3gOkLA/g/IqDQL/QgTq7/3eloHaO+uPBihdF56do4eaOO0jgFYpl8V7ek
PZhHfhuPZV3oq15+8Vt77ngtjUWVI6qX0E3ilh+V5cof+03q0FzHPVe3zBUNXcm0
OGz19u/FAoGBAPSm4Aa4xs/ybyjQakMNix9rak66ehzGkmlfeK5yuQ/fHmTg8Ac+
ahGs6A3lFWQiyU6hqm6Qp0iKuxuDh35DJGCWAw5OUS/7WLJtu8fNFch6iIG29rFs
s+Uz2YLxJPebpBsKymZUp7NyDRgEElkiqsREmbYjLrc8uNKkDy+k14YnAoGBAPGn
ZlN0Mo5iNgQStulYEP5pI7WOOax9KOYVnBNguqgY9c7fXVXBxChoxt5ebQJWG45y
KPG0hB0bkA4YPu4bTRf5acIMpjFwcxNlmwdc4oCkT4xqAFs9B/AKYZgkf4IfKHqW
P9PD7TbUpkaxv25bPYwUSEB7lPa+hBtRyN9Wo6qfAoGAPBkeISiU1hJE0i7YW55h
FZfKZoqSYq043B+ywo+1/Dsf+UH0VKM1ZSAnZPpoVc/hyaoW9tAb98r0iZ620wJl
VkCjgYklknbY5APmw/8SIcxP6iVq1kzQqDYjcXIRVa3rEyWEcLzM8VzL8KFXbIQC
lPIRHFfqKuMEt+HLRTXmJ7MCgYAHGvv4QjdmVl7uObqlG9DMGj1RjlAF0VxNf58q
NrLmVG2N2qV86wigg4wtZ6te4TdINfUcPkmQLYpLz8yx5Z2bsdq5OPP+CidoD5nC
WqnSTIKGR2uhQycjmLqL5a7WHaJsEFTqHh2wego1k+5kCUzC/KmvM7MKmkl6ICp+
3qZLUwKBgQCDOhKDwYo1hdiXoOOQqg/LZmpWOqjO3b4p99B9iJqhmXN0GKXIPSBh
5nqqmGsG8asSQhchs7EPMh8B80KbrDTeidWskZuUoQV27Al1UEmL6Zcl83qXD6sf
k9X9TwWyZtp5IL1CAEd/Il9ZTXFzr3lNaN8LCFnU+EIsz1YgUW8LTg==
-----END RSA PRIVATE KEY-----
`)

// NewLocalListener returns a new ipv4 "127.0.0.1:0"
// or tcp6 "[::1]:0" tcp listener.
func NewLocalListener() net.Listener {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if l, err = net.Listen("tcp6", "[::1]:0"); err != nil {
			panic(err)
		}
	}
	return l
}

// NewLocalTLSListener returns a new tls listener
// based on the "tcpListener", if "tcpListener" is nil
// it make use of the `NewLocalListener`.
// Cert and Key are `LocalhostCert` and `LocalhostKey` respectfully.
func NewLocalTLSListener(tcpListener net.Listener) net.Listener {
	if tcpListener == nil {
		tcpListener = NewLocalListener()
	}

	cert, err := tls.X509KeyPair(LocalhostCert, LocalhostKey)
	if err != nil {
		panic(err)
	}

	cfg := new(tls.Config)
	cfg.NextProtos = []string{"http/1.1"}
	cfg.Certificates = []tls.Certificate{cert}
	cfg.InsecureSkipVerify = true
	return tls.NewListener(tcpListener, cfg)
}
