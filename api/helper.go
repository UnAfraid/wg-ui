package api

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/UnAfraid/wg-ui/api/model"
	"github.com/UnAfraid/wg-ui/internal/adapt"
	"github.com/UnAfraid/wg-ui/server"
)

func idsToStringIds(idKind model.IdKind, ids []*model.ID) ([]string, error) {
	return adapt.ArrayErr(ids, func(id *model.ID) (string, error) {
		return id.String(idKind)
	})
}

func (r *resolverRoot) withServer(ctx context.Context, serverId string, callback func(svc *server.Server)) error {
	svc, err := r.serverService.FindServer(ctx, &server.FindOneOptions{
		IdOption: &server.IdOption{
			Id: serverId,
		},
		NameOption: nil,
	})
	if err != nil {
		return err
	}
	if svc != nil {
		callback(svc)
	}
	return nil
}

func checkOrigin(r *http.Request, allowedSubscriptionOrigins []string) bool {
	origin := r.Header["Origin"]
	if len(origin) == 0 {
		return true
	}

	u, err := url.Parse(origin[0])
	if err != nil {
		return false
	}

	for _, allowedHost := range allowedSubscriptionOrigins {
		allowedHost = strings.TrimSpace(allowedHost)
		if allowedHost == "*" {
			return true
		}
		if equalASCIIFold(u.Host, allowedHost) {
			return true
		}
	}

	return equalASCIIFold(u.Host, r.Host)
}

func equalASCIIFold(s, t string) bool {
	for s != "" && t != "" {
		sr, size := utf8.DecodeRuneInString(s)
		s = s[size:]
		tr, size := utf8.DecodeRuneInString(t)
		t = t[size:]
		if sr == tr {
			continue
		}
		if 'A' <= sr && sr <= 'Z' {
			sr = sr + 'a' - 'A'
		}
		if 'A' <= tr && tr <= 'Z' {
			tr = tr + 'a' - 'A'
		}
		if sr != tr {
			return false
		}
	}
	return s == t
}
