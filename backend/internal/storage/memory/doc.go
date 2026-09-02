// Package memory holds in-memory implementations of the Store interfaces that
// the domain packages (internal/account, …) declare. It is the default store
// for local development and for tests — no database, no network.
//
// A sibling internal/storage/sqlite package provides real persistence and is
// added through its own OpenSpec change. package main picks one implementation
// at startup and injects it into each domain service.
//
// This package is intentionally empty until the first domain package lands and
// declares a Store to implement.
package memory
