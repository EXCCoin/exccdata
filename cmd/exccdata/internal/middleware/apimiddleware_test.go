// Copyright (c) 2019-2021, The Decred developers
// See LICENSE for details.

package middleware

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/EXCCoin/exccd/chaincfg/v3"
	"github.com/go-chi/chi/v5"
)

func TestGetAddressCtx(t *testing.T) {
	activeNetParams := chaincfg.MainNetParams()
	type args struct {
		maxAddrs int
		addrs    []string
	}
	tests := []struct {
		testName string
		args     args
		want     []string
		wantErr  bool
		errMsg   string
		wantCode int
	}{
		{
			testName: "ok2",
			args:     args{2, []string{"22u4yVXyXzZjqpXwvnmNBf1J8vQVvF6xZu5S"}},
			want:     []string{"22u4yVXyXzZjqpXwvnmNBf1J8vQVvF6xZu5S"},
			wantErr:  false,
		},
		{
			testName: "ok1",
			args:     args{1, []string{"22u4yVXyXzZjqpXwvnmNBf1J8vQVvF6xZu5S"}},
			want:     []string{"22u4yVXyXzZjqpXwvnmNBf1J8vQVvF6xZu5S"},
			wantErr:  false,
		},
		{
			testName: "bad0",
			args:     args{0, []string{"22u4yVXyXzZjqpXwvnmNBf1J8vQVvF6xZu5S"}},
			want:     nil, // not []string{}
			wantErr:  true,
			errMsg:   "maximum of 0 addresses allowed",
			wantCode: http.StatusUnprocessableEntity,
		},
		{
			testName: "bad3",
			args: args{2, []string{"22u4yVXyXzZjqpXwvnmNBf1J8vQVvF6xZu5S",
				"22tki4qR9DjPEKdmT8wEjuE4YxexWPrZS3TH",
				"22u38ScQ2df45ppzQ4CrNbxjMsDZrbWK3SKh"}},
			want:     nil,
			wantErr:  true,
			errMsg:   "maximum of 2 addresses allowed",
			wantCode: http.StatusUnprocessableEntity,
		},
		{
			// This tests that the middleware counts before removing dups.
			testName: "bad_dup3",
			args: args{2, []string{"22u38ScQ2df45ppzQ4CrNbxjMsDZrbWK3SKh",
				"22tki4qR9DjPEKdmT8wEjuE4YxexWPrZS3TH",
				"22u38ScQ2df45ppzQ4CrNbxjMsDZrbWK3SKh"}},
			want:     nil,
			wantErr:  true,
			errMsg:   "maximum of 2 addresses allowed",
			wantCode: http.StatusUnprocessableEntity,
		},
		{
			// This tests that the middleware counts removes dups.
			testName: "ok_dup3",
			args: args{3, []string{"22u38ScQ2df45ppzQ4CrNbxjMsDZrbWK3SKh",
				"22tki4qR9DjPEKdmT8wEjuE4YxexWPrZS3TH",
				"22u38ScQ2df45ppzQ4CrNbxjMsDZrbWK3SKh"}},
			want: []string{"22u38ScQ2df45ppzQ4CrNbxjMsDZrbWK3SKh",
				"22tki4qR9DjPEKdmT8wEjuE4YxexWPrZS3TH"},
			wantErr: false,
		},
		{
			testName: "invalid",
			args:     args{2, []string{"22xxxxxxxxjPEKdmT8wEjuE4YxexWPrZS3TH"}},
			want:     nil,
			wantErr:  true,
			errMsg:   `invalid address "22xxxxxxxxjPEKdmT8wEjuE4YxexWPrZS3TH" for this network: failed to decoded address "22xxxxxxxxjPEKdmT8wEjuE4YxexWPrZS3TH": checksum error`},
		{
			testName: "wrong_net",
			args:     args{2, []string{"TsbTSs4HZXkNo1MRXFEQqTwgCdPdp2Cy2bZ"}},
			want:     nil,
			wantErr:  true,
			errMsg:   `invalid address "TsbTSs4HZXkNo1MRXFEQqTwgCdPdp2Cy2bZ" for this network: address "TsbTSs4HZXkNo1MRXFEQqTwgCdPdp2Cy2bZ" is not a supported type`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			router := chi.NewRouter()
			tAddrCtx := AddressPathCtxN(tt.args.maxAddrs)
			var run bool
			router.With(tAddrCtx).Get("/{address}", func(w http.ResponseWriter, r *http.Request) {
				run = true
				got, err := GetAddressCtx(r, activeNetParams) //, tt.args.maxAddrs)
				if (err != nil) != tt.wantErr {
					t.Errorf("GetAddressCtx() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("GetAddressCtx() = %v, want %v", got, tt.want)
				}
				if err != nil && tt.errMsg != err.Error() {
					t.Fatalf(`GetAddressCtx() error = "%v", expected "%s"`, err, tt.errMsg)
				}
			})
			writer := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/"+strings.Join(tt.args.addrs, ","), nil)
			router.ServeHTTP(writer, req)
			switch tt.wantCode {
			case 0, 200:
				if !run {
					t.Errorf("handler not reached")
				}
				if writer.Code != 200 {
					t.Errorf("expected 200, got %d", writer.Code)
				}
			default:
				if tt.wantCode != writer.Code {
					t.Errorf("expected response code %d, got %d", tt.wantCode, writer.Code)
				}
			}
		})
	}
}
