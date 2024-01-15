package api

import (
	"context"
	"k8s-update-deployment-ecr-tag/webhook/api/handler"
	"net/http"
)

type App struct {
}

func (app *App) HandleMutate(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	respAdmissionReview, error := handler.Handler(ctx, r)
	if error != nil {
		jsonError(w, error.Error(), http.StatusInternalServerError)
	}
	jsonOk(w, &respAdmissionReview)
}
