---
name: Bug report
about: Create a report to help us improve
title: "[BUG]"
labels: type:bug
assignees: kataras

---

**Describe the bug**
A clear and concise description of what the bug is.

**To Reproduce**
Steps to reproduce the behavior:
1. [...]

**Expected behavior**
A clear and concise description of what you expected to happen.

**Screenshots**
If applicable, add screenshots to help explain your problem.

**Desktop (please complete the following information):**
 - OS: [e.g. ubuntu, windows]

**iris.Version**
- e.g. v12.2.5 or main

Please make sure the bug is reproducible over the `main` branch:

```sh
$ cd PROJECT
$ go get -u github.com/kataras/iris/v12@main
$ go run .
```

**Additional context**
Add any other context about the problem here.
