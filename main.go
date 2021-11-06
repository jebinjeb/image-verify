package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/julienschmidt/httprouter"
	"github.com/sigstore/cosign/pkg/cosign"
	"github.com/sigstore/cosign/pkg/cosign/fulcio"
	ociremote "github.com/sigstore/cosign/pkg/oci/remote"
)

// RegistryOptions is the wrapper for the registry options.
type RegistryOptions struct {
	AllowInsecure      bool
	KubernetesKeychain bool
	RefOpts            ReferenceOptions
}

type ImageVerificationReq struct {
	Image string
}

type ImageVerificationResp struct {
	Verified            bool   `json:"verified"`
	VerificationMessage string `json:"verification_message"`
}

func Verify(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var body ImageVerificationReq
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := context.TODO()

	wDir, err := os.Getwd()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	key, err := cosign.LoadPublicKey(ctx, filepath.Join(wDir, "cosign.pub"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ref, err := name.ParseReference(body.Image)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	co := &cosign.CheckOpts{
		RootCerts:   fulcio.Roots,
		SigVerifier: key,
		Claims:      true,
	}

	var resp ImageVerificationResp
	if _, err = cosign.Verify(ctx, ref, co); err != nil {
		resp = ImageVerificationResp{
			Verified:            false,
			VerificationMessage: err.Error(),
		}
		log.Println(resp)
	} else {
		resp = ImageVerificationResp{
			Verified:            true,
			VerificationMessage: fmt.Sprintf("valid signatures found for an image: %s", body.Image),
		}
		log.Println(resp)
	}

	respAsByte, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(respAsByte)
}

func main() {
	router := httprouter.New()
	router.POST("/verify", Verify)
	log.Fatal(http.ListenAndServe(":8080", router))
}

func (o *RegistryOptions) ClientOpts(ctx context.Context) ([]ociremote.Option, error) {
	opts := []ociremote.Option{ociremote.WithRemoteOptions(o.GetRegistryClientOpts(ctx)...)}
	if o.RefOpts.TagPrefix != "" {
		opts = append(opts, ociremote.WithPrefix(o.RefOpts.TagPrefix))
	}
	targetRepoOverride, err := ociremote.GetEnvTargetRepository()
	if err != nil {
		return nil, err
	}
	if (targetRepoOverride != name.Repository{}) {
		opts = append(opts, ociremote.WithTargetRepository(targetRepoOverride))
	}
	return opts, nil
}