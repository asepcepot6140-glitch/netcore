package routes

import (
	"time"

	"github.com/asepcepot6140-glitch/netcore/internal/controllers"
	"github.com/asepcepot6140-glitch/netcore/internal/controllers/auth"
	"github.com/asepcepot6140-glitch/netcore/internal/controllers/genieacs"
	"github.com/asepcepot6140-glitch/netcore/internal/controllers/hotspot"
	"github.com/asepcepot6140-glitch/netcore/internal/controllers/iface"
	"github.com/asepcepot6140-glitch/netcore/internal/controllers/pppoe"
	"github.com/asepcepot6140-glitch/netcore/internal/controllers/simplequeue"
	"github.com/asepcepot6140-glitch/netcore/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	api := r.Group("/api")

	// Batas per IP untuk menahan brute force dan spam OTP WhatsApp.
	authLimit := middleware.RateLimit(10, time.Minute)
	otpLimit := middleware.RateLimit(5, time.Minute)

	// Batas per nomor telepon agar serangan dari banyak IP tetap terbatas per akun target.
	// verify-otp dan reset-password berbagi satu kuota.
	otpRequestPhoneLimit := middleware.RateLimitByPhone(5, time.Hour)
	otpGuessPhoneLimit := middleware.RateLimitByPhone(10, 15*time.Minute)

	// Public Auth
	api.POST("/register", authLimit, auth.Register)
	api.POST("/login", authLimit, auth.Login)
	api.POST("/refresh", authLimit, auth.Refresh)

	api.POST("/forgot-password", otpLimit, otpRequestPhoneLimit, auth.ForgotPassword)
	api.POST("/verify-otp", otpLimit, otpGuessPhoneLimit, auth.VerifyOTP)
	api.POST("/reset-password", otpLimit, otpGuessPhoneLimit, auth.ResetPassword)

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
	protected.GET("/pppoe/live", pppoe.Live)
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
	protected.GET("/pppoe/queues", pppoe.ListQueueNames)

	// PPPoE Offline
	protected.GET("/pppoe/offline", pppoe.ListOfflineConnections)

	// Hotspot Users CRUD
	protected.GET("/hotspot/live", hotspot.Live)
	protected.GET("/hotspot/users", hotspot.ListUsers)
	protected.GET("/hotspot/users/:id", hotspot.ShowUser)
	protected.POST("/hotspot/users", hotspot.CreateUser)
	protected.PUT("/hotspot/users/:id", hotspot.UpdateUser)
	protected.DELETE("/hotspot/users/:id", hotspot.DeleteUser)

	// Hotspot User Profiles CRUD
	protected.GET("/hotspot/profiles", hotspot.ListProfiles)
	protected.GET("/hotspot/profiles/:id", hotspot.ShowProfile)
	protected.POST("/hotspot/profiles", hotspot.CreateProfile)
	protected.PUT("/hotspot/profiles/:id", hotspot.UpdateProfile)
	protected.DELETE("/hotspot/profiles/:id", hotspot.DeleteProfile)

	// Hotspot Active (online) and Hosts
	protected.GET("/hotspot/servers", hotspot.ListServers)
	protected.GET("/hotspot/pools", hotspot.ListPools)
	protected.GET("/hotspot/queues", hotspot.ListQueues)
	protected.GET("/hotspot/queue-types", hotspot.ListQueueTypes)
	protected.GET("/hotspot/active", hotspot.ListActive)
	protected.DELETE("/hotspot/active/:id", hotspot.KickActive)
	protected.GET("/hotspot/hosts", hotspot.ListHosts)
	protected.DELETE("/hotspot/hosts/:id", hotspot.DeleteHost)

	// Simple Queue CRUD
	protected.GET("/queue/simple/live", simplequeue.Live)
	protected.GET("/queue/simple", simplequeue.List)
	protected.POST("/queue/simple", simplequeue.Create)
	protected.PUT("/queue/simple/:id", simplequeue.Update)
	protected.DELETE("/queue/simple/:id", simplequeue.Delete)
	protected.GET("/queue/types", simplequeue.ListTypes)
	protected.GET("/queue/parents", simplequeue.ListParents)
	protected.GET("/queue/packet-marks", simplequeue.ListPacketMarks)

	// Interface list, enable/disable, and edit
	protected.GET("/interfaces", iface.List)
	protected.GET("/interfaces/live", iface.Live)
	protected.GET("/interfaces/:id/monitor", iface.Monitor)
	protected.PUT("/interfaces/:id", iface.Update)

	protected.GET("/genieacs", genieacs.List)
	protected.POST("/genieacs", genieacs.Create)
	protected.GET("/genieacs/:id", genieacs.Show)
	protected.PUT("/genieacs/:id", genieacs.Update)
	protected.DELETE("/genieacs/:id", genieacs.Delete)

	protected.GET("/devices/:id", genieacs.ListDevices)
	protected.GET("/devices/:id/:deviceId/wan", genieacs.GetWAN)
	protected.GET("/devices/:id/:deviceId/lan", genieacs.GetLAN)
	protected.GET("/devices/:id/:deviceId/wlan", genieacs.GetWLAN)
	protected.POST("/devices/:id/:deviceId/summon", genieacs.SummonDevice)
	protected.DELETE("/devices/:id/:deviceId", genieacs.DeleteDevice)
	protected.POST("/devices/:id/:deviceId/wan", genieacs.AddWAN)
	protected.PUT("/devices/:id/:deviceId/wan", genieacs.UpdateWAN)
	protected.DELETE("/devices/:id/:deviceId/wan", genieacs.DeleteWAN)
	protected.PUT("/devices/:id/:deviceId/lan/dhcp", genieacs.UpdateLANDHCP)
	protected.PUT("/devices/:id/:deviceId/lan/ports", genieacs.UpdateLANPort)
	protected.PUT("/devices/:id/:deviceId/wlan", genieacs.UpdateWLAN)
}
