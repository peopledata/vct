/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

//go:generate mockgen -destination gomocks_test.go -self_package mocks -package vct_test . HTTPClient

package vct_test

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hyperledger/aries-framework-go/pkg/doc/util"
	"github.com/hyperledger/aries-framework-go/pkg/doc/verifiable"
	"github.com/stretchr/testify/require"

	"github.com/trustbloc/vct/pkg/canonicalizer"
	"github.com/trustbloc/vct/pkg/client/vct"
	"github.com/trustbloc/vct/pkg/controller/command"
	"github.com/trustbloc/vct/pkg/controller/rest"
	"github.com/trustbloc/vct/pkg/testutil"
)

const endpoint = "https://example.com"

//go:embed testdata/bachelor_degree.json
var vcBachelorDegree []byte // nolint: gochecknoglobals

func TestClient_AddVC(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		expected := command.AddVCResponse{
			SVCTVersion: 1,
			ID:          []byte(`id`),
			Timestamp:   1234567889,
			Extensions:  "extensions",
			Signature:   []byte(`signature`),
		}

		expectedCredential := []byte(`{credential}`)

		fakeResp, err := json.Marshal(expected)
		require.NoError(t, err)

		httpClient := NewMockHTTPClient(ctrl)
		httpClient.EXPECT().Do(gomock.Any()).Do(func(req *http.Request) {
			var credential []byte

			credential, err = ioutil.ReadAll(req.Body)
			require.NoError(t, err)
			require.Equal(t, expectedCredential, credential)
		}).Return(&http.Response{
			Body:       ioutil.NopCloser(bytes.NewBuffer(fakeResp)),
			StatusCode: http.StatusOK,
		}, nil)

		client := vct.New(endpoint, vct.WithHTTPClient(httpClient), vct.WithAuthReadToken("tk1"),
			vct.WithAuthWriteToken("tk2"))
		resp, err := client.AddVC(context.Background(), expectedCredential)
		require.NoError(t, err)

		bytesResp, err := json.Marshal(resp)
		require.NoError(t, err)

		require.Equal(t, fakeResp, bytesResp)
	})

	t.Run("Error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		expected := rest.ErrorResponse{Message: "error"}

		fakeResp, err := json.Marshal(expected)
		require.NoError(t, err)

		httpClient := NewMockHTTPClient(ctrl)
		httpClient.EXPECT().Do(gomock.Any()).Return(&http.Response{
			Body:       ioutil.NopCloser(bytes.NewBuffer(fakeResp)),
			StatusCode: http.StatusInternalServerError,
		}, nil)

		client := vct.New(endpoint, vct.WithHTTPClient(httpClient))
		_, err = client.AddVC(context.Background(), []byte{})
		require.EqualError(t, err, "add VC: error")
	})
}

func TestClient_HealthCheck(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		httpClient := NewMockHTTPClient(ctrl)
		httpClient.EXPECT().Do(gomock.Any()).Do(func(req *http.Request) {
		}).Return(&http.Response{
			Body:       ioutil.NopCloser(bytes.NewBuffer([]byte("{}"))),
			StatusCode: http.StatusOK,
		}, nil)

		client := vct.New("https://vct:22/maple2020", vct.WithHTTPClient(httpClient), vct.WithAuthReadToken("tk1"))
		err := client.HealthCheck(context.Background())
		require.NoError(t, err)
	})

	t.Run("Error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		httpClient := NewMockHTTPClient(ctrl)
		httpClient.EXPECT().Do(gomock.Any()).Do(func(req *http.Request) {
		}).Return(&http.Response{
			Body:       ioutil.NopCloser(bytes.NewBuffer([]byte("{}"))),
			StatusCode: http.StatusInternalServerError,
		}, nil)

		client := vct.New(endpoint, vct.WithHTTPClient(httpClient), vct.WithAuthReadToken("tk1"))
		err := client.HealthCheck(context.Background())
		require.Error(t, err)
	})
}

func TestClient_GetIssuers(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		expected := []string{"issuer_1", "issuer_2"}

		fakeResp, err := json.Marshal(expected)
		require.NoError(t, err)

		httpClient := NewMockHTTPClient(ctrl)
		httpClient.EXPECT().Do(gomock.Any()).Return(&http.Response{
			Body:       ioutil.NopCloser(bytes.NewBuffer(fakeResp)),
			StatusCode: http.StatusOK,
		}, nil)

		client := vct.New(endpoint, vct.WithHTTPClient(httpClient))
		resp, err := client.GetIssuers(context.Background())
		require.NoError(t, err)

		bytesResp, err := json.Marshal(resp)
		require.NoError(t, err)

		require.Equal(t, fakeResp, bytesResp)
	})

	t.Run("Error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		expected := rest.ErrorResponse{Message: "error"}

		fakeResp, err := json.Marshal(expected)
		require.NoError(t, err)

		httpClient := NewMockHTTPClient(ctrl)
		httpClient.EXPECT().Do(gomock.Any()).Return(&http.Response{
			Body:       ioutil.NopCloser(bytes.NewBuffer(fakeResp)),
			StatusCode: http.StatusInternalServerError,
		}, nil)

		client := vct.New(endpoint, vct.WithHTTPClient(httpClient))
		_, err = client.GetIssuers(context.Background())
		require.EqualError(t, err, "get issuers: error")
	})
}

func TestClient_Webfinger(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		expected := command.WebFingerResponse{
			Subject: "https://vct.com/maple2021",
			Properties: map[string]interface{}{
				"https://trustbloc.dev/ns/public-key": "cHVibGljIGtleQ==",
			},
			Links: []command.WebFingerLink{{
				Rel:  "self",
				Href: "https://vct.com/maple2021",
			}},
		}

		fakeResp, err := json.Marshal(expected)
		require.NoError(t, err)

		httpClient := NewMockHTTPClient(ctrl)
		httpClient.EXPECT().Do(gomock.Any()).Return(&http.Response{
			Body:       ioutil.NopCloser(bytes.NewBuffer(fakeResp)),
			StatusCode: http.StatusOK,
		}, nil)

		client := vct.New(endpoint, vct.WithHTTPClient(httpClient))
		resp, err := client.Webfinger(context.Background())
		require.NoError(t, err)

		bytesResp, err := json.Marshal(resp)
		require.NoError(t, err)

		require.Equal(t, fakeResp, bytesResp)
	})

	t.Run("Error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		expected := rest.ErrorResponse{Message: "error"}

		fakeResp, err := json.Marshal(expected)
		require.NoError(t, err)

		httpClient := NewMockHTTPClient(ctrl)
		httpClient.EXPECT().Do(gomock.Any()).Return(&http.Response{
			Body:       ioutil.NopCloser(bytes.NewBuffer(fakeResp)),
			StatusCode: http.StatusInternalServerError,
		}, nil)

		client := vct.New(endpoint, vct.WithHTTPClient(httpClient))
		_, err = client.Webfinger(context.Background())
		require.EqualError(t, err, "webfinger: error")
	})
}

func TestClient_GetSTH(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		expected := command.GetSTHResponse{
			TreeSize:          1,
			Timestamp:         1234567889,
			SHA256RootHash:    []byte(`SHA256RootHash`),
			TreeHeadSignature: []byte(`TreeHeadSignature`),
		}

		fakeResp, err := json.Marshal(expected)
		require.NoError(t, err)

		httpClient := NewMockHTTPClient(ctrl)
		httpClient.EXPECT().Do(gomock.Any()).Return(&http.Response{
			Body:       ioutil.NopCloser(bytes.NewBuffer(fakeResp)),
			StatusCode: http.StatusOK,
		}, nil)

		client := vct.New(endpoint, vct.WithHTTPClient(httpClient))
		resp, err := client.GetSTH(context.Background())
		require.NoError(t, err)

		bytesResp, err := json.Marshal(resp)
		require.NoError(t, err)

		require.Equal(t, fakeResp, bytesResp)
	})

	t.Run("Error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		expected := rest.ErrorResponse{Message: "error"}

		fakeResp, err := json.Marshal(expected)
		require.NoError(t, err)

		httpClient := NewMockHTTPClient(ctrl)
		httpClient.EXPECT().Do(gomock.Any()).Return(&http.Response{
			Body:       ioutil.NopCloser(bytes.NewBuffer(fakeResp)),
			StatusCode: http.StatusInternalServerError,
		}, nil)

		client := vct.New(endpoint, vct.WithHTTPClient(httpClient))
		_, err = client.GetSTH(context.Background())
		require.EqualError(t, err, "get STH: error")
	})
}

func TestClient_GetSTHConsistency(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		expected := command.GetSTHConsistencyResponse{
			Consistency: [][]byte{[]byte("consistency")},
		}

		fakeResp, err := json.Marshal(expected)
		require.NoError(t, err)

		httpClient := NewMockHTTPClient(ctrl)
		httpClient.EXPECT().Do(gomock.Any()).Do(func(req *http.Request) {
			require.Equal(t, "1", req.URL.Query().Get("first"))
			require.Equal(t, "2", req.URL.Query().Get("second"))
		}).Return(&http.Response{
			Body:       ioutil.NopCloser(bytes.NewBuffer(fakeResp)),
			StatusCode: http.StatusOK,
		}, nil)

		client := vct.New(endpoint, vct.WithHTTPClient(httpClient))
		resp, err := client.GetSTHConsistency(context.Background(), 1, 2)
		require.NoError(t, err)

		bytesResp, err := json.Marshal(resp)
		require.NoError(t, err)

		require.Equal(t, fakeResp, bytesResp)
	})

	t.Run("Error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		expected := rest.ErrorResponse{Message: "error"}

		fakeResp, err := json.Marshal(expected)
		require.NoError(t, err)

		httpClient := NewMockHTTPClient(ctrl)
		httpClient.EXPECT().Do(gomock.Any()).Return(&http.Response{
			Body:       ioutil.NopCloser(bytes.NewBuffer(fakeResp)),
			StatusCode: http.StatusInternalServerError,
		}, nil)

		client := vct.New(endpoint, vct.WithHTTPClient(httpClient))
		_, err = client.GetSTHConsistency(context.Background(), 1, 2)
		require.EqualError(t, err, "get STH consistency: error")
	})
}

func TestClient_GetProofByHash(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		expected := command.GetProofByHashResponse{
			LeafIndex: 1,
			AuditPath: [][]byte{[]byte("audit path")},
		}

		fakeResp, err := json.Marshal(expected)
		require.NoError(t, err)

		httpClient := NewMockHTTPClient(ctrl)
		httpClient.EXPECT().Do(gomock.Any()).Do(func(req *http.Request) {
			require.Equal(t, "hash", req.URL.Query().Get("hash"))
			require.Equal(t, "2", req.URL.Query().Get("tree_size"))
		}).Return(&http.Response{
			Body:       ioutil.NopCloser(bytes.NewBuffer(fakeResp)),
			StatusCode: http.StatusOK,
		}, nil)

		client := vct.New(endpoint, vct.WithHTTPClient(httpClient))
		resp, err := client.GetProofByHash(context.Background(), "hash", 2)
		require.NoError(t, err)

		bytesResp, err := json.Marshal(resp)
		require.NoError(t, err)

		require.Equal(t, fakeResp, bytesResp)
	})

	t.Run("Error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		expected := rest.ErrorResponse{Message: "error"}

		fakeResp, err := json.Marshal(expected)
		require.NoError(t, err)

		httpClient := NewMockHTTPClient(ctrl)
		httpClient.EXPECT().Do(gomock.Any()).Return(&http.Response{
			Body:       ioutil.NopCloser(bytes.NewBuffer(fakeResp)),
			StatusCode: http.StatusInternalServerError,
		}, nil)

		client := vct.New(endpoint, vct.WithHTTPClient(httpClient))
		_, err = client.GetProofByHash(context.Background(), "hash", 2)
		require.EqualError(t, err, "get proof by hash: error")
	})
}

func TestClient_GetEntries(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		expected := command.GetEntriesResponse{
			Entries: []command.LeafEntry{{LeafInput: []byte(`leaf input`), ExtraData: []byte(`extra data`)}},
		}

		fakeResp, err := json.Marshal(expected)
		require.NoError(t, err)

		httpClient := NewMockHTTPClient(ctrl)
		httpClient.EXPECT().Do(gomock.Any()).Do(func(req *http.Request) {
			require.Equal(t, "1", req.URL.Query().Get("start"))
			require.Equal(t, "2", req.URL.Query().Get("end"))
		}).Return(&http.Response{
			Body:       ioutil.NopCloser(bytes.NewBuffer(fakeResp)),
			StatusCode: http.StatusOK,
		}, nil)

		client := vct.New(endpoint, vct.WithHTTPClient(httpClient))
		resp, err := client.GetEntries(context.Background(), 1, 2)
		require.NoError(t, err)

		bytesResp, err := json.Marshal(resp)
		require.NoError(t, err)

		require.Equal(t, fakeResp, bytesResp)
	})

	t.Run("Error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		expected := rest.ErrorResponse{Message: "error"}

		fakeResp, err := json.Marshal(expected)
		require.NoError(t, err)

		httpClient := NewMockHTTPClient(ctrl)
		httpClient.EXPECT().Do(gomock.Any()).Return(&http.Response{
			Body:       ioutil.NopCloser(bytes.NewBuffer(fakeResp)),
			StatusCode: http.StatusInternalServerError,
		}, nil)

		client := vct.New(endpoint, vct.WithHTTPClient(httpClient))
		_, err = client.GetEntries(context.Background(), 1, 2)
		require.EqualError(t, err, "get entries: error")
	})
}

func TestClient_GetEntryAndProof(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		expected := command.GetEntryAndProofResponse{
			LeafInput: []byte(`leaf input`),
			ExtraData: []byte(`extra data`),
			AuditPath: [][]byte{[]byte(`audit path`)},
		}

		fakeResp, err := json.Marshal(expected)
		require.NoError(t, err)

		httpClient := NewMockHTTPClient(ctrl)
		httpClient.EXPECT().Do(gomock.Any()).Do(func(req *http.Request) {
			require.Equal(t, "1", req.URL.Query().Get("leaf_index"))
			require.Equal(t, "2", req.URL.Query().Get("tree_size"))
		}).Return(&http.Response{
			Body:       ioutil.NopCloser(bytes.NewBuffer(fakeResp)),
			StatusCode: http.StatusOK,
		}, nil)

		client := vct.New(endpoint, vct.WithHTTPClient(httpClient))
		resp, err := client.GetEntryAndProof(context.Background(), 1, 2)
		require.NoError(t, err)

		bytesResp, err := json.Marshal(resp)
		require.NoError(t, err)

		require.Equal(t, fakeResp, bytesResp)
	})

	t.Run("Error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		expected := rest.ErrorResponse{Message: "error"}

		fakeResp, err := json.Marshal(expected)
		require.NoError(t, err)

		httpClient := NewMockHTTPClient(ctrl)
		httpClient.EXPECT().Do(gomock.Any()).Return(&http.Response{
			Body:       ioutil.NopCloser(bytes.NewBuffer(fakeResp)),
			StatusCode: http.StatusInternalServerError,
		}, nil)

		client := vct.New(endpoint, vct.WithHTTPClient(httpClient))
		_, err = client.GetEntryAndProof(context.Background(), 1, 2)
		require.EqualError(t, err, "get entry and proof: error")
	})
}

var simpleVC = &verifiable.Credential{ // nolint: gochecknoglobals // global vc
	Context: []string{"https://www.w3.org/2018/credentials/v1"},
	Subject: "did:key:123",
	Issuer:  verifiable.Issuer{ID: "did:key:123"},
	Issued: func() *util.TimeWrapper {
		res := &util.TimeWrapper{}

		json.Unmarshal([]byte("\"2020-03-10T04:24:12.164Z\""), &res) // nolint: errcheck, gosec

		return res
	}(),
	Types:  []string{"VerifiableCredential"},
	Proofs: []verifiable.Proof{{}, {}},
}

var simpleVC2 = &verifiable.Credential{ // nolint: gochecknoglobals // global vc
	Context: []string{"https://www.w3.org/2018/credentials/v1", "https://w3id.org/security/suites/ed25519-2020/v1"},
	Subject: "did:key:123",
	Issuer:  verifiable.Issuer{ID: "did:key:123"},
	Issued: func() *util.TimeWrapper {
		res := &util.TimeWrapper{}

		json.Unmarshal([]byte("\"2020-03-10T04:24:12.164Z\""), &res) // nolint: errcheck, gosec

		return res
	}(),
	Types:  []string{"VerifiableCredential"},
	Proofs: []verifiable.Proof{{"xxx": "yyy"}},
}

func TestCalculateLeafHash(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		vcBytes1, err := json.Marshal(simpleVC)
		require.NoError(t, err)

		hash1, err := vct.CalculateLeafHash(12345, vcBytes1, testutil.GetLoader(t))
		require.NoError(t, err)
		require.NotEmpty(t, hash1)

		// Should be the same hash even if the VC has additional contexts, different prroofs, and
		// is marshalled in a different way.
		vcBytes2, err := canonicalizer.MarshalCanonical(simpleVC2)
		require.NoError(t, err)

		hash2, err := vct.CalculateLeafHash(12345, vcBytes2, testutil.GetLoader(t))
		require.NoError(t, err)
		require.Equal(t, hash1, hash2)
	})
}

func TestVerifyVCTimestampSignature(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		const signature = `{
  "algorithm": {
    "signature": "ECDSA",
    "type": "ECDSAP256DER"
  },
  "signature": "MEUCIQCBCoNVefPQCbfp/v7XBbd8bW1FeE4tRXnY2m2HRECyMAIgWoaG8Bz9pLIewVRLlzym5svZ+YKp2i9yv+2uk/CRBjo="
}`

		pubKey, err := base64.StdEncoding.DecodeString(
			"MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEYH7+MO+X0YPnGkvK1Nmy/4/r9HpgPPku9gjw3k3zOl+PTbu7iEL2gsiH/KHaFbeMoMcj5Tv0OkA/EKfuzd0imQ==") //nolint:lll
		require.NoError(t, err)

		require.NoError(t, vct.VerifyVCTimestampSignature(
			[]byte(signature), pubKey, 1662067083140, vcBachelorDegree, testutil.GetLoader(t),
		))
	})

	t.Run("Unmarshal signature error", func(t *testing.T) {
		require.Contains(t, vct.VerifyVCTimestampSignature(
			[]byte(`[]`), []byte(`[]`), 1617977793917, vcBachelorDegree, testutil.GetLoader(t),
		).Error(), "unmarshal signature")
	})

	t.Run("Wrong public key", func(t *testing.T) {
		require.Contains(t, vct.VerifyVCTimestampSignature(
			[]byte(`{}`), []byte(`[]`), 1617977793917, vcBachelorDegree, testutil.GetLoader(t),
		).Error(), "pub key to handle: error")
	})
}
