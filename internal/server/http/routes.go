package http

import (
	"github.com/gin-gonic/gin"
	"github.com/purpledb/purple/internal/server/http/handler"
)

func (s *Server) routes() *gin.Engine {
	r := gin.New()

	r.GET("/ping", s.h.Ping)

	cache := r.Group("/cache/:key")
	{
		cache.GET("", s.h.CacheGet)

		withTtl := cache.Group("")
		{
			withTtl.Use(handler.SetTtl)
			withTtl.PUT("", s.h.CachePut)
		}
	}

	counters := r.Group("/counters/:key")
	{
		counters.GET("", s.h.CounterGet)

		withIncr := counters.Group("")
		{
			withIncr.Use(handler.SetIncr)
			withIncr.PUT("", s.h.CounterPut)
		}
	}

	flags := r.Group("/flags/:key")
	{
		flags.GET("", s.h.FlagGet)

		withVal := flags.Group("")
		{
			withVal.Use(handler.SetFlagValue)
			withVal.PUT("", s.h.FlagSet)
		}
	}

	kv := r.Group("/kv/:key")
	{
		kv.GET("", s.h.KvGet)
		kv.DELETE("", s.h.KvDelete)

		withVal := kv.Group("")
		{
			withVal.Use(handler.SetKVValue)
			withVal.PUT("", s.h.KvPut)
		}
	}

	sets := r.Group("/sets/:key")
	{
		sets.GET("", s.h.SetGet)

		withItem := sets.Group("")
		{
			withItem.Use(handler.SetItem)
			withItem.PUT("", s.h.SetPut)
			withItem.DELETE("", s.h.SetDelete)
		}
	}

	return r
}
