# Contributing

Contributions keep Agent OS adopter-clean. Public docs, examples, tests, and release material use
public artifact coordinates only.

## Before Opening a Pull Request

1. Keep install and update examples on released public artifacts.
2. Do not present a local source checkout, local source substitution, or private source path as an
   adopter install or update path.
3. Keep compatibility facts in component-owned contracts and release metadata.
4. Use the catalog only as discovery and cross-reference material.
5. Run the unit gate:

```sh
GOTOOLCHAIN=local bash scripts/verify.sh --unit
```

## Code And Docs Standard

- Changes are scoped to one logical purpose.
- Names describe the local responsibility they own.
- Tests protect behavior, not the current file layout.
- Docs state future gates when an artifact is not yet publicly released.

## License

By contributing, you agree that your contribution is licensed under Apache-2.0.
