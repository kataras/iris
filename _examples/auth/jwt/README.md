# Generate RSA

```sh
$ openssl genrsa -des3 -out private_rsa.pem 2048
```

```go
b, err := ioutil.ReadFile("./private_rsa.pem")
if err != nil {
    panic(err)
}
key := jwt.MustParseRSAPrivateKey(b, []byte("pass"))
```

OR

```go
import "crypto/rand"
import "crypto/rsa"

key, err := rsa.GenerateKey(rand.Reader, 2048)
```

# Generate Ed25519

```sh
$ openssl genpkey -algorithm Ed25519 -out private_ed25519.pem
$ openssl req -x509 -key private_ed25519.pem -out cert_ed25519.pem -days 365
```
