# Project working rules

- Work only on the explicitly requested lesson task and mentor comments.
- Keep every change minimal; do not add refactoring, abstractions, tooling, or behavior that is not required.
- Follow the lesson's examples and required scenario when choosing API, names, and structure.
- One commit must address one purpose and contain no more than 300 changed lines.
- One HTTP endpoint has one handler file and one corresponding test file.
- Before adding a CRUD endpoint, identify the business process it represents; name endpoints and methods after that process.
- Name service and repository methods after the domain action they perform, not generic technical operations such as `Update`.
- Run the smallest relevant check after each change, then run the required project-wide checks before handing off.
