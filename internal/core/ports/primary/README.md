# Primary Ports (Driving Ports)

Primary ports define **what the outside world can ask the application to do**.

These represent the **use cases** of your application from the perspective of external actors (users, other systems, CLI, tests, etc.).

## Purpose

- Declare the public API of the application
- Use business language (not technical terms)
- Remain completely technology-agnostic

## Example Use Case

```go
type UserService interface {
    Register(ctx context.Context, cmd RegisterUserCommand) (User, error)
    GetProfile(ctx context.Context, id UserID) (UserProfile, error)
}
```
