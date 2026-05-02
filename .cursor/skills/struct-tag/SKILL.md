---
name: go-struct-tag
description: Validates Go struct tags for serialization
---

# Go Struct Tag Skill

When reviewing or writing Go code:
- Ensure JSON tags match exported field names (snake_case or camelCase)
- Validate YAML tags are present where needed
- Check that database tags match column names
- Ensure omitempty is used appropriately
- Validate that required fields don't have omitempty
- Check for duplicate or conflicting tags
- Ensure tag values are properly quoted
- Validate that tag keys are lowercase
- Check for custom tag validation where needed
- Ensure struct tags are consistent across the codebase
