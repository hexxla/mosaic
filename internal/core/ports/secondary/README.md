# Secondary Ports (Driven Ports)

Secondary ports define **what the application needs from the outside world** to fulfill its use cases.

These are outbound contracts that the core depends on.

## Purpose

- Allow the core to remain decoupled from concrete implementations
- Expressed from the application's point of view
- Implemented by secondary adapters

## Example

```go
type UserRepository interface {
    Save(ctx context.Context, user *User) error
    FindByID(ctx context.Context, id UserID) (*User, error)
    ExistsByEmail(ctx context.Context, email string) (bool, error)
}
```
