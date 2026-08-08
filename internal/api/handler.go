package api

import (
	"github.com/gospelfast/gospelfast/internal/cache"
	"github.com/gospelfast/gospelfast/internal/db"
)

type Handler struct {
	DB    *db.DB
	Cache *cache.Cache
}

func NewHandler(d *db.DB, c *cache.Cache) *Handler {
	return &Handler{DB: d, Cache: c}
}
