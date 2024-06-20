package main

const TEST string = `
---
name: test

when:
  branch:
    include:
      - main
  env TEST4:
    match:
      - ^.xam.[le]{2}$

steps:
  - name: test
    task: docker
    command: [sh]
    input: |
      echo Hello, $test!
      echo How are you?

      wget -O - "$REEVE_API/api/v1/var?key=a&value=test"
    params:
      test:
        env: TEST
        replace:
          - /ex/😉s/
    when:
      branch:
        include:
          - main
      env TEST3:
        match:
          - ^.xam.[le]{2}$
      env TEST99:
        include env:
          - TEST100

  - name: skip-me
    task: docker
    command: sh -c "echo hello, $test - $test2"
    params:
      test: { env: TEST5 }
      test2: { var: a }
    when:
      branch:
        include:
          - main
      env TEST6:
        match:
          - ^.xam.[le]{2}$
      var a:
        exclude:
          - test
`
