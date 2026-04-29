# Hexagonal Architecture - Optimal Design Flow

**A step-by-step guide to designing systems using hexagonal architecture**

---

## The Golden Rule: Work Inside-Out

Always design from the **center outward**:

```
Domain → Primary Ports → Secondary Ports → Services → Adapters
```

This ensures your core business logic drives the design, not technical choices like databases or frameworks.

---

## Step 1: Define Business Concepts (Domain)

**Start here.** Everything else depends on this.

### What to do

- Talk to stakeholders to understand the business
- Identify the "things" (entities) in the business
- Define business rules and invariants
- Create value objects for common concepts

### Questions to ask

- What are the nouns in this business? (User, Order, Product, Session)
- What are the rules? (User must have unique email, Order must have at least one item)
- What are the shared concepts? (Email, Money, DateRange)

### Output

```go
// internal/core/domain/user.go
package domain

import "errors"

// UserID is a value object representing a unique user identifier
type UserID string

// Email is a value object representing a validated email address
type Email string

func (e Email) Validate() error {
    // Email validation logic
    return nil
}

// User is a domain entity representing a user in the system
type User struct {
    ID           UserID
    Email        Email
    PasswordHash string
    IsActive     bool
}

// Activate is a business rule - a user can only be activated once
func (u *User) Activate() error {
    if u.IsActive {
        return errors.New("user already active")
    }
    u.IsActive = true
    return nil
}
```

### Domain Methods vs Service Methods

**Entity methods (business rules) are defined on entities in `core/domain/`:**

- `user.go` contains the `User` entity struct
- Methods like `User.Activate()` are business rules about what the entity CAN do
- These methods operate on the entity itself
- Self-contained, no external dependencies
- Examples: `User.Activate()`, `User.ValidatePassword()`, `User.IsEmailValid()`

**Service methods (use cases) are defined in `core/services/`:**

- `user_service.go` contains use case implementations
- Methods like `Register()` coordinate what the system DOES
- They call entity methods, repositories, email senders, etc.
- Examples: `Register()`, `Login()`, `ResetPassword()`

**Example:**

- `User.Activate()` (entity method in domain) = "User can only be activated once" - business rule
- `UserService.Register()` (service method) = "Check email exists, create user, save, send email" - use case

### Why first?

- Everything depends on domain
- If you get this wrong, everything breaks
- Domain is the most stable part of the system

---

## Step 2: Define Use Cases (Primary Ports)

**What can the outside world ask the service layer to do?**

**Note:** Primary ports are interfaces (contracts), not implementations. Sometimes called "outside world ports" or "driving ports". The service layer will implement these interfaces.

### What to do

- Identify use cases from stakeholder requirements
- Define interfaces for each use case
- Use business language, not technical terms
- Keep interfaces focused and cohesive

### Questions to ask

- What do users want to do? (Register, Login, CreateOrder, CancelOrder)
- What do other systems want to do? (SyncUser, GetOrderStatus)
- What are the commands and queries?

### Output

```go
// internal/core/ports/primary/user_service.go
package port

import (
    "context"

    "github.com/myapp/internal/core/domain"
)

// UserService is an interface (port) defining what the outside world can ask the service layer to do
// The service layer will implement this interface
type UserService interface {
    // A user wants to sign up
    Register(ctx context.Context, cmd RegisterUserCommand) (domain.User, error)

    // A user wants to see their profile
    GetProfile(ctx context.Context, id domain.UserID) (UserProfile, error)

    // A user wants to change their name
    UpdateName(ctx context.Context, id domain.UserID, name string) error
}

// RegisterUserCommand is a command object for user registration
type RegisterUserCommand struct {
    Email    domain.Email
    Password string
    Name     string
}
```

### Why second?

- Defines the contract with the outside world
- Forces you to think about what the system should do before building it
- Primary ports depend on domain, so domain must exist first

---

## Step 3: Define External Dependencies (Secondary Ports)

**What external systems do we need to interact with?**

**Note:** "Outside world" here means databases, APIs, and other services - not users. Users interact through primary ports. Secondary ports are sometimes called "infrastructure ports", "dependency ports", or "driven ports".

### What to do

- Identify what external systems the application needs to interact with
- Define interfaces for each dependency
- Express from the application's point of view
- Don't think about implementation details yet

### Questions to ask

- Where do we store data? Define a port like `UserRepository` (interface). The adapter will implement it for PostgreSQL, MongoDB, etc.
- Do we send emails? Define a port like `EmailSender`. The adapter will implement it for SMTP, SendGrid, etc.
- Do we call APIs? Define a port like `PaymentGateway`. The adapter will implement it for Stripe, PayPal, etc.
- Do we need caching? Define a port like `Cache`. The adapter will implement it for Redis, Memcached, etc.

### Output

```go
// internal/core/ports/secondary/user_repository.go
package port

import (
    "context"

    "github.com/myapp/internal/core/domain"
)

// UserRepository defines what the application needs for data persistence
type UserRepository interface {
    Save(ctx context.Context, user *domain.User) error
    FindByID(ctx context.Context, id domain.UserID) (*domain.User, error)
    ExistsByEmail(ctx context.Context, email domain.Email) (bool, error)
}

// internal/core/ports/secondary/email_sender.go
package port

import (
    "context"

    "github.com/myapp/internal/core/domain"
)

// EmailSender defines what the application needs for sending emails
type EmailSender interface {
    SendWelcomeEmail(ctx context.Context, to domain.Email) error
    SendPasswordReset(ctx context.Context, to domain.Email, token string) error
}
```

### Why third?

- Defines what functionality the service layer needs from adapters
- Secondary ports depend on domain, so domain must exist first
- Services will need these interfaces, so define them before implementing services
- Adapters will implement these to connect to actual external systems (PostgreSQL, SMTP, etc.)

---

## Step 4: Implement Services

**Wire it together.**

**Note:** Services depend on secondary port interfaces (defined in Step 3) to know what methods are available to call. Adapters will implement these interfaces later.

### What to do

- Implement primary port interfaces
- Use domain entities for business logic
- Call secondary ports for external operations
- Keep services focused on orchestration, not business rules

### Questions to ask

- How do I coordinate the use case?
- What domain entities do I need?
- Which secondary ports do I call?
- In what order?

### Output

```go
// internal/core/services/user_service.go
package service

import (
    "context"
    "errors"

    "github.com/myapp/internal/core/domain"
    "github.com/myapp/internal/core/ports"
)

// UserServiceImpl implements the UserService port interface
type UserServiceImpl struct {
    userRepo    port.UserRepository
    emailSender port.EmailSender
}

func NewUserService(userRepo port.UserRepository, emailSender port.EmailSender) *UserServiceImpl {
    return &UserServiceImpl{
        userRepo:    userRepo,
        emailSender: emailSender,
    }
}

func (s *UserServiceImpl) Register(ctx context.Context, cmd port.RegisterUserCommand) (domain.User, error) {
    // 1. Check if email exists (secondary port)
    exists, err := s.userRepo.ExistsByEmail(ctx, cmd.Email)
    if err != nil {
        return domain.User{}, err
    }
    if exists {
        return domain.User{}, errors.New("email already taken")
    }

    // 2. Create user (domain logic)
    user := domain.User{
        Email: cmd.Email,
        Name:  cmd.Name,
    }

    // 3. Save user (secondary port)
    err = s.userRepo.Save(ctx, &user)
    if err != nil {
        return domain.User{}, err
    }

    // 4. Send welcome email (secondary port)
    s.emailSender.SendWelcomeEmail(ctx, user.Email)

    return user, nil
}
```

### Why fourth?

- Pure business logic, easy to test with mocks
- Depends on domain and ports, which are already defined
- Can be fully tested before choosing databases or frameworks

---

## Step 5: Implement Adapters

**Last - connect to the real world.**

### What to do

- Implement primary port interfaces for external requests
- Implement secondary port interfaces for external systems
- Choose your technologies (HTTP framework, database, etc.)
- Handle technical concerns (serialization, error mapping)

### Questions to ask

- What protocol do we use? (HTTP/gRPC for primary, SQL/HTTP for secondary)
- What framework? (Gin, gRPC, sqlx, GORM)
- How do we convert between protocol and domain types?

### Output

```go
// internal/adapter/primary/http_user_handler.go
package primary

import (
    "encoding/json"
    "net/http"

    "github.com/myapp/internal/core/domain"
    "github.com/myapp/internal/core/ports"
)

type HTTPUserHandler struct {
    userService port.UserService
}

func NewHTTPUserHandler(userService port.UserService) *HTTPUserHandler {
    return &HTTPUserHandler{userService: userService}
}

type RegisterRequest struct {
    Email    string `json:"email"`
    Password string `json:"password"`
    Name     string `json:"name"`
}

func (h *HTTPUserHandler) Register(w http.ResponseWriter, r *http.Request) {
    var req RegisterRequest
    json.NewDecoder(r.Body).Decode(&req)

    cmd := port.RegisterUserCommand{
        Email:    domain.Email(req.Email),
        Password: req.Password,
        Name:     req.Name,
    }

    user, err := h.userService.Register(r.Context(), cmd)
    // Handle response...
    _ = user // Use user
    _ = err  // Handle error
}

// internal/adapter/secondary/postgres_user_repository.go
package secondary

import (
    "context"
    "database/sql"

    "github.com/myapp/internal/core/domain"
    "github.com/myapp/internal/core/ports"
)

type PostgresUserRepository struct {
    db *sql.DB
}

func NewPostgresUserRepository(db *sql.DB) *PostgresUserRepository {
    return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) Save(ctx context.Context, user *domain.User) error {
    _, err := r.db.ExecContext(ctx,
        "INSERT INTO users (id, email, name) VALUES ($1, $2, $3)",
        user.ID, user.Email, user.Name,
    )
    return err
}

func (r *PostgresUserRepository) FindByID(ctx context.Context, id domain.UserID) (*domain.User, error) {
    var user domain.User
    err := r.db.QueryRowContext(ctx, "SELECT id, email, name FROM users WHERE id = $1", id).
        Scan(&user.ID, &user.Email, &user.Name)
    if err != nil {
        return nil, err
    }
    return &user, nil
}

func (r *PostgresUserRepository) ExistsByEmail(ctx context.Context, email domain.Email) (bool, error) {
    var exists bool
    err := r.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", email).
        Scan(&exists)
    return exists, err
}

// Ensure PostgresUserRepository implements port.UserRepository
var _ port.UserRepository = (*PostgresUserRepository)(nil)
```

### Why last?

- Most changeable part (swap Postgres for Mongo, HTTP for gRPC)
- Depends on ports, which are already defined
- Technical choice shouldn't drive business logic

---

## Complete Example: User Registration Flow

### Step 1: Domain (entity with business rule methods)

```go
// internal/core/domain/user.go
type User struct {
    ID    UserID
    Email Email
    Name  string
}

type Email string

// Business rule method on the entity
func (u *User) Activate() error { ... }
```

### Step 2: Primary Port (interface, not implementation)

```go
// internal/core/ports/primary/user_service.go
type UserService interface {
    Register(ctx context.Context, cmd RegisterUserCommand) (User, error)
}
```

### Step 3: Secondary Ports (interfaces for external dependencies)

```go
// internal/core/ports/secondary/user_repository.go
type UserRepository interface {
    Save(ctx context.Context, user *User) error
    ExistsByEmail(ctx context.Context, email string) (bool, error)
}

// internal/core/ports/secondary/email_sender.go
type EmailSender interface {
    SendWelcomeEmail(ctx context.Context, to Email) error
}
```

### Step 4: Service (implements primary port, coordinates use case)

```go
// internal/core/services/user_service.go
type UserServiceImpl struct {
    userRepo    UserRepository  // Secondary port interface
    emailSender EmailSender      // Secondary port interface
}

// Implements UserService interface (primary port)
func (s *UserServiceImpl) Register(ctx context.Context, cmd RegisterUserCommand) (User, error) {
    // Coordinate: check email, create entity, call entity methods, save, send email
    exists, _ := s.userRepo.ExistsByEmail(ctx, cmd.Email)
    user := User{Email: cmd.Email, Name: cmd.Name}
    s.userRepo.Save(ctx, &user)
    s.emailSender.SendWelcomeEmail(ctx, user.Email)
    return user, nil
}
```

### Step 5: Adapters (implement secondary ports, connect to real world)

```go
// Primary adapter (HTTP) - calls primary port
type HTTPHandler struct {
    userService UserService  // Primary port interface
}

func (h *HTTPHandler) Register(w http.ResponseWriter, r *http.Request) {
    // Parse HTTP request, call userService.Register
}

// Secondary adapter (PostgreSQL) - implements secondary port
type PostgresUserRepository struct {
    db *sql.DB
}

// Implements UserRepository interface (secondary port)
func (r *PostgresUserRepository) Save(ctx context.Context, user *User) error {
    // INSERT INTO users ...
}
```

---

## Benefits of This Approach

| Benefit            | How This Flow Helps                                           |
| ------------------ | ------------------------------------------------------------- |
| **Testable**       | Core logic can be tested with mocks before choosing databases |
| **Flexible**       | Swap adapters without touching core logic                     |
| **Clear**          | Business concepts drive design, not technical choices         |
| **Parallelizable** | Different people can work on different layers simultaneously  |
| **Documented**     | Ports serve as living documentation of contracts              |

---

## Common Mistakes to Avoid

### Starting with Adapters

**Wrong:** "I'll use PostgreSQL and Gin, now let me build around them"
**Right:** "I need to store users and handle HTTP, let me define ports first"

### Mixing Layers

**Wrong:** Service directly importing adapter
**Right:** Service depending on port, adapter implementing port

### Business Logic in Wrong Place

**Wrong:** Validation in HTTP handler or SQL constraints
**Right:** Business rules as methods on domain entities (e.g., `User.Activate()`)

### Skipping Ports

**Wrong:** Service directly using sql.DB
**Right:** Service using UserRepository interface, adapter implements it

---

## Quick Reference

| Order | Layer           | Output                                                    | Why This Order                      |
| ----- | --------------- | --------------------------------------------------------- | ----------------------------------- |
| 1     | Domain          | Entities with business rule methods, value objects        | Everything depends on this          |
| 2     | Primary Ports   | Use case interfaces (contracts)                           | Defines contract with outside world |
| 3     | Secondary Ports | Dependency interfaces (contracts)                         | Defines what app needs from outside |
| 4     | Services        | Use case implementations that coordinate entities & ports | Pure logic, easy to test            |
| 5     | Adapters        | HTTP, gRPC, database implementations of ports             | Most changeable, do last            |

---

## When to Deviate

This flow is a guideline, not a rigid rule. Deviate when:

- **Prototyping:** Skip strict layering for quick experiments
- **Legacy integration:** May need to start with existing adapters
- **Simple CRUD:** Domain might be trivial, start with ports
- **Framework-driven:** Some frameworks encourage adapter-first thinking

But always return to inside-out thinking for production code.
