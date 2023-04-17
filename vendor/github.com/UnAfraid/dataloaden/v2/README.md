### DataLoader Generator ![Go](https://github.com/UnAfraid/dataloaden/workflows/Go/badge.svg) [![Go Report Card](https://goreportcard.com/badge/github.com/UnAfraid/dataloaden)](https://goreportcard.com/report/github.com/UnAfraid/dataloaden)

For Go 1.18 using Generics use v2

#### Getting started

From inside the package you want to have the dataloader in:

```bash
go run github.com/UnAfraid/dataloaden/v2 -name UserLoader -fileName generated_user_loader.go -keyType string -valueType *github.com/UnAfraid/dataloaden/v2/example.User
```

This will generate a dataloader called `UserLoader` that looks up `*github.com/UnAfraid/dataloaden/v2/example.User`'s objects based on a `string` key.

Then wherever you want to call the dataloader

```go
loader := NewUserLoader(dataloader.Config[string, *User]{
    Fetch: func (keys []string) ([]*User, []error) {
        users := make([]*User, len(keys))
        errors := make([]error, len(keys))
        
        for i, key := range keys {
          users[i] = &User{ID: key, Name: "user " + key}
        }
        return users, errors
    },
    Wait:     2 * time.Millisecond,
    MaxBatch: 100,
})

user, err := loader.Load("123")
```

---

Requires golang 1.16+ for modules support.

This is a tool for generating type safe data loaders for go, inspired by https://github.com/facebook/dataloader.

The intended use is in graphql servers, to reduce the number of queries being sent to the database. These dataloader objects should be request scoped and short-lived. They should be cheap to create in every request even
if they don't get used.

#### Getting started

From inside the package you want to have the dataloader in:

```bash
go run github.com/UnAfraid/dataloaden -name UserLoader -fileName generated_user_loader.go -keyType string -valueType *github.com/UnAfraid/dataloaden/example.User
```

This will generate a dataloader called `UserLoader` that looks up `*github.com/UnAfraid/dataloaden/example.User`'s objects based on a `string` key.

Then wherever you want to call the dataloader

```go
loader := NewUserLoader(UserLoaderConfig{
    Fetch: func (keys []string) ([]*User, []error) {
        users := make([]*User, len(keys))
        errors := make([]error, len(keys))
        
        for i, key := range keys {
          users[i] = &User{ID: key, Name: "user " + key}
        }
        return users, errors
    },
    Wait:     2 * time.Millisecond,
    MaxBatch: 100,
})

user, err := loader.Load("123")
```

This method will block for a short amount of time, waiting for any other similar requests to come in, call your fetch function once. It also caches values and wont request duplicates in a batch.

#### Returning Slices

You may want to generate a dataloader that returns slices instead of single values. Both key and value types can be a simple go type expression:

```bash
go run github.com/UnAfraid/dataloaden -name UserSliceLoader -keyType string -valueType []*github.com/dataloaden/example.User
```

Now each key is expected to return a slice of values and the `fetch` function has the return type `[][]*User`.

#### Using with go modules

Create a tools.go that looks like this:

```go
// +build tools

package main

import _ "github.com/UnAfraid/dataloaden"
```

This will allow go modules to see the dependency.

You can invoke it from anywhere within your module now using `go run github.com/UnAfraid/dataloaden` and always get the pinned version.

#### Wait, how do I use context with this?

I don't think context makes sense to be passed through a data loader. Consider a few scenarios:

1. a dataloader shared between requests: request A and B both get batched together, which context should be passed to the DB? context.Background is probably more suitable.
2. a dataloader per request for graphql: two different nodes in the graph get batched together, they have different context for tracing purposes, which should be passed to the db? neither, you should just use the root
   request context.

So be explicit about your context:

```go
NewUserLoader(UserLoaderConfig{
    Fetch: func (keys []string) ([]*User, []error) {
        // you now have a ctx to work with
    },
    Wait:     2 * time.Millisecond,
    MaxBatch: 100,
})
```

If you feel like I'm wrong please raise an issue.
