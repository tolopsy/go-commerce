package urlsigner

import (
	"fmt"
	"strings"
	"time"

	goalone "github.com/bwmarrin/go-alone"
)

type Signer struct {
	secret []byte
}

func NewSigner(secret []byte) *Signer {
	return &Signer{
		secret: secret,
	}
}
func (s *Signer) GenerateTokenFromString(unsignedURL string) string {
	var urlToSign string
	crypt := goalone.New(s.secret, goalone.Timestamp)

	if strings.Contains(unsignedURL, "?") {
		urlToSign = fmt.Sprintf("%s&hash=", unsignedURL)
	} else {
		urlToSign = fmt.Sprintf("%s?hash=", unsignedURL)
	}

	return string(crypt.Sign([]byte(urlToSign)))
}

func (s *Signer) VerifyToken(token string) bool {
	crypt := goalone.New(s.secret, goalone.Timestamp)
	if _, err := crypt.Unsign([]byte(token)); err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

func (s *Signer) Expired(token string, minutesUntilExpire int) bool {
	crypt := goalone.New(s.secret, goalone.Timestamp)
	ts := crypt.Parse([]byte(token))

	return time.Since(ts.Timestamp) > time.Duration(minutesUntilExpire) * time.Minute
}