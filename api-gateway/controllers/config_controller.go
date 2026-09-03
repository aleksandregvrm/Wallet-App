package controllers

import (
	"net/http"
)

type ConfigController struct{}

func NewConfigController() *ConfigController {
	return &ConfigController{}
}

func (c *ConfigController) HandleHealthLiveness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "live"}`))
}

func (c *ConfigController) HandleHealthReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "ready"}`))
}
