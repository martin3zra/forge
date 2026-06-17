# Routing Package for Go

A lightweight HTTP routing library built on Go’s standard `net/http` package. This package gives you powerful route management driven by simple, expressive APIs. It supports dynamic URL parameters, query and JSON binding, global and group‑specific middleware (with options to skip certain middleware), and robust request handling across multiple HTTP methods (GET, POST, PUT, DELETE). The library now supports Laravel‑style route grouping with dedicated prefixes and nested groups.

## Features

- **Route Registration:**
  - Register routes easily with HTTP methods like GET, POST, PUT, DELETE.
  - Define dynamic segments (e.g., `/users/:id`) that automatically extract URL parameters.

- **Middleware Support:**
  - Apply global middleware via `WithMiddleware`.
  - Chain and wrap middleware in the order they were registered.
  - Exclude specific middleware per route using `WithoutMiddleware`.
  - Remove middleware for an entire router (or group) using `WithoutNextMiddleware`.

- **Route Grouping and Prefixing:**
  - **Group with Prefix:**
    The new API allows you to group routes under a common prefix with a dedicated method called `GroupPrefix`. This method temporarily updates the router’s prefix for the duration of the group callback and restores it afterward—providing a clean, Laravel‑like API without spawning separate router instances.

    For example, you can now write:
    ```go
    // Create a group with a prefix:
    r.GroupPrefix("admin", func(admin *routing.Router) {
        admin.GET("users", func(ctx *routing.Context) {
            // Matches "/admin/users"
        })
    })

    // Nested groups are supported:
    r.GroupPrefix("api", func(api *routing.Router) {
        api.GroupPrefix("v1", func(v1 *routing.Router) {
            v1.GET("resource", resourceHandler) // Matches "/api/v1/resource"
        })
        api.GET("status", statusHandler) // Matches "/api/status"
    })

    // Middleware Exclusion Example:
    r.GroupPrefix("blog", func(blog *routing.Router) {
        // Exclude a specific middleware from this route.
        blog.GET("post", postHandler).WithoutMiddleware(authMiddleware)
    })
    ```

    In these examples, notice that the sub‑route definitions use relative paths (without a leading slash) so that they concatenate correctly with the group’s prefix. If a group is created with an empty prefix, the behavior remains unchanged.

  - **Group Without Prefix:**
    Standard grouping (using `Group(fn)`) continues to be available when no additional path modification is needed. Group‑specific middleware can also be applied in this fashion.

- **Data Binding:**
  - Built‑in support for binding query parameters (using `BindQuery`).
  - Built‑in JSON payload binding (using `BindJSON`) for processing request bodies.

- **Routing Robustness:**
  - Automatic path normalization removes trailing slashes (except for the root).
  - Provides 405 (Method Not Allowed) responses when a path is found but the HTTP method does not match.
  - Comprehensive dynamic route matching, ensuring robust route resolution.

## Installation

Simply copy the `pkg/routing` directory into your project. Since this is a lightweight package with no external dependencies (aside from the Go standard library), you can integrate it directly into your codebase.

## Basic Usage

Below is a sample application demonstrating how to create a router, register routes, add middleware, use the new clean prefix‑based grouping API, nest groups, and skip middleware when needed.

```go
package main

import (
    "fmt"
    "log"
    "net/http"

    "your_module/pkg/routing" // Update this import path as needed.
)

func main() {
    // Create a new router.
    r := routing.New()

    // Global middleware example:
    logger := func(next routing.HandlerFunc) routing.HandlerFunc {
        return func(ctx *routing.Context) {
            fmt.Println("Received request for", ctx.Request.URL.Path)
            next(ctx)
        }
    }
    r.WithMiddleware(logger)

    // Basic route registration.
    r.GET("/hello", func(ctx *routing.Context) {
        ctx.JSON(http.StatusOK, map[string]string{"message": "Hello, world!"})
    })

    // Dynamic route with URL parameter.
    r.GET("/users/:id", func(ctx *routing.Context) {
        userID := ctx.Params["id"]
        ctx.Text(http.StatusOK, "User ID: " + userID)
    })

    // Data binding: Query and JSON.
    r.GET("/search", func(ctx *routing.Context) {
        type SearchQuery struct {
            Term string `query:"term"`
        }
        var q SearchQuery
        if err := ctx.BindQuery(&q); err != nil {
            ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
            return
        }
        ctx.JSON(http.StatusOK, map[string]string{"searched": q.Term})
    })

    r.POST("/register", func(ctx *routing.Context) {
        type User struct {
            Name  string `json:"name"`
            Email string `json:"email"`
        }
        var user User
        if err := ctx.BindJSON(&user); err != nil {
            ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
            return
        }
        ctx.JSON(http.StatusOK, user)
    })

    // Extra middleware examples.
    authMiddleware := func(next routing.HandlerFunc) routing.HandlerFunc {
        return func(ctx *routing.Context) {
            token := ctx.Request.Header.Get("Authorization")
            if token != "secret" {
                ctx.Text(http.StatusUnauthorized, "Unauthorized")
                return
            }
            next(ctx)
        }
    }

    metricsMiddleware := func(next routing.HandlerFunc) routing.HandlerFunc {
        return func(ctx *routing.Context) {
            // Imagine logging metrics here.
            ctx.Response.Header().Set("X-Metrics", "recorded")
            next(ctx)
        }
    }

    // Global routes can also be registered normally.
    r.GET("/public", func(ctx *routing.Context) {
        ctx.Text(http.StatusOK, "Public content")
    })

    // Group routes under "/api" with nested grouping and selective middleware.
    r.WithMiddleware(authMiddleware) // Global auth middleware.
    r.GroupPrefix("api", func(api *routing.Router) {
        // "/api/secure" route inherits auth middleware.
        api.GET("secure", func(ctx *routing.Context) {
            ctx.Text(http.StatusOK, "Secure API Content")
        })
        // "/api/public" route skips auth middleware.
        api.GET("public", func(ctx *routing.Context) {
            ctx.Text(http.StatusOK, "Public API Content")
        }).WithoutMiddleware(authMiddleware)
    })

    // Nested group example for admin routes.
    r.GroupPrefix("admin", func(admin *routing.Router) {
        admin.GET("dashboard", func(ctx *routing.Context) {
            ctx.Text(http.StatusOK, "Admin Dashboard")
        })
        // Nested group within admin.
        admin.GroupPrefix("settings", func(settings *routing.Router) {
            settings.GET("profile", func(ctx *routing.Context) {
                ctx.Text(http.StatusOK, "Profile Settings")
            })
        })
    })

    // List all registered routes (useful for debugging).
    for _, route := range r.Routes() {
        fmt.Println("Registered route:", route)
    }

    fmt.Println("Server starting on port :8080")
    log.Fatal(http.ListenAndServe(":8080", r))
}
