package tools

import (
	"encoding/json"
	"os"

	"github.com/emicklei/go-restful"
	"github.com/gin-gonic/gin"
)

type R struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}
type M struct {
	Name   string `json:"name,omitempty"`
	Routes []R    `json:"routes,omitempty"`
}

func Routes(name string, cc *restful.Container) {
	m := M{Name: name, Routes: make([]R, 0, 8)}
	services := cc.RegisteredWebServices()
	for _, service := range services {
		for _, route := range service.Routes() {
			m.Routes = append(m.Routes, R{
				Method: route.Method,
				Path:   route.Path,
			})
		}
	}
	marshal, err := json.Marshal(m)
	if err != nil {
		return
	}
	os.WriteFile(name+".txt", marshal, 0644)
}

func RoutesX(name string, cc *gin.Engine) {
	m := M{Name: name, Routes: make([]R, 0, 8)}
	for _, service := range cc.Routes() {
		m.Routes = append(m.Routes, R{
			Method: service.Method,
			Path:   service.Path,
		})
	}
	marshal, err := json.Marshal(m)
	if err != nil {
		return
	}
	os.WriteFile(name+".txt", marshal, 0644)
}
