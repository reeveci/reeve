package main

const TEST2 string = `
---
name: test2

steps:
  - name: test2
    task: docker
    command: { env: COMMAND }
`
