package routes

import (
	"github.com/asepcepot6140-glitch/netcore/internal/controllers"
	"github.com/asepcepot6140-glitch/netcore/internal/controllers/auth"
	"github.com/asepcepot6140-glitch/netcore/internal/controllers/genieacs"
	"github.com/asepcepot6140-glitch/netcore/internal/controllers/pppoe"
	"github.com/asepcepot6140-glitch/netcore/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	api := r.Group("/api")

	// Public Auth
	api.POST("/register", auth.Register)
	api.POST("/login", auth.Login)
	api.POST("/refresh", auth.Refresh)

	api.POST("/forgot-password", auth.ForgotPassword)
	api.POST("/verify-otp", auth.VerifyOTP)
	api.POST("/reset-password", auth.ResetPassword)

	// Protected
	protected := api.Group("")
	protected.Use(middleware.Auth())

	protected.GET("/me", auth.Me)
	protected.POST("/logout", auth.Logout)

	// Mikrotik CRUD
	protected.GET("/mikrotiks", controllers.ListMikrotik)
	protected.POST("/mikrotiks", controllers.CreateMikrotik)
	protected.GET("/mikrotiks/:id", controllers.ShowMikrotik)
	protected.PUT("/mikrotiks/:id", controllers.UpdateMikrotik)
	protected.DELETE("/mikrotiks/:id", controllers.DeleteMikrotik)

	// PPPoE Secrets CRUD
	protected.GET("/pppoe/secrets", pppoe.ListSecrets)
	protected.GET("/pppoe/secrets/:id", pppoe.ShowSecret)
	protected.POST("/pppoe/secrets", pppoe.CreateSecret)
	protected.PUT("/pppoe/secrets/:id", pppoe.UpdateSecret)
	protected.DELETE("/pppoe/secrets/:id", pppoe.DeleteSecret)

	// PPP Profiles CRUD
	protected.GET("/pppoe/profiles", pppoe.ListProfiles)
	protected.GET("/pppoe/profiles/:name", pppoe.ShowProfile)
	protected.POST("/pppoe/profiles", pppoe.CreateProfile)
	protected.PUT("/pppoe/profiles/:name", pppoe.UpdateProfile)
	protected.DELETE("/pppoe/profiles/:name", pppoe.DeleteProfile)

	// PPP Active
	protected.GET("/pppoe/active", pppoe.ListActiveConnections)
	protected.DELETE("/pppoe/active/:id", pppoe.RemoveActiveConnection)

	// PPP Pools
	protected.GET("/pppoe/pools", pppoe.ListPools)

	// PPPoE Offline
	protected.GET("/pppoe/offline", pppoe.ListOfflineConnections)

	protected.GET("/genieacs", genieacs.List)
	protected.POST("/genieacs", genieacs.Create)
	protected.GET("/genieacs/:id", genieacs.Show)
	protected.PUT("/genieacs/:id", genieacs.Update)
	protected.DELETE("/genieacs/:id", genieacs.Delete)

	protected.GET("/devices/:id", genieacs.ListDevices)
	protected.POST("/devices/:id/:deviceId/summon", genieacs.SummonDevice)
}
