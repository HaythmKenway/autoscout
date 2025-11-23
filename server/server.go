package server

// func graphqlHandler() gin.HandlerFunc {
// 	h := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: &graph.Resolver{}}))

// 	return func(c *gin.Context) {
// 		h.ServeHTTP(c.Writer, c.Request)
// 	}
// }

// func playgroundHandler() gin.HandlerFunc {
// 	h := playground.Handler("GraphQL", "/query")
// 	return func(c *gin.Context) {
// 		h.ServeHTTP(c.Writer, c.Request)
// 	}
// }

// // func Server() {
// // 	localUtils.Logger("Starting Gin Server", 1)

// //		r := gin.Default()
// //		r.Use(corsMiddleware())
// //		r.POST("/query", graphqlHandler())
// //		r.GET("/", playgroundHandler())
// //		r.Run(":8000")
// //	}
// func corsMiddleware() gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
// 		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
// 		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

// 		if c.Request.Method == "OPTIONS" {
// 			c.AbortWithStatus(204)
// 			return
// 		}

// 		c.Next()
// 	}
// }
