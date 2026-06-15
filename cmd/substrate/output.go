package main

import (
	"github.com/pay-bye/agent-os/internal/declaration"
	"io"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func writeDelta(out io.Writer, delta declaration.Delta) error {
	content, err := declaration.Render(delta)
	if err != nil {
		return err
	}
	_, err = out.Write(content)
	return err
}

func help() string {
	return `substrate

The substrate sits quiet until commanded.

commands:
  serve                 run the HTTP boundary using explicit process inputs
  credential generate   print one bearer key and verifier digest, then exit
  init                  write a declaration file template
  preview               read configured inputs and print Registry vocabulary changes
  apply                 read configured inputs and mutate Registry vocabulary

operator key hygiene:
  Treat every raw key as bearer authority. Store raw keys outside the substrate.
  Rotate operator authority by replacing the verifier document and restarting serve.
  Operator-Key gates operator instruction routes.`
}

func serveHelp() string {
	return `serve

Purpose:
  Runs the HTTP boundary for submit, claim, ack, nack, extend, heartbeat, compatibility, probes, metrics, and operations.

Inputs:
  --database-url <url>             database connection string
  --listen <address>               listener address
  --from <path>                    declaration file
  --config <path>                  process config file
  --verifier-digest <digest>       boundary credential verifier digest
  --verifier-file <path>           boundary credential verifier document
  --operator-verifier-file <path>  operator verifier document
  --shutdown-grace <duration>      graceful shutdown bound

Operator verifier document:
  {"algorithm":"sha256-base64url","digest":"<verifier_digest>"}
  The file must be a regular file with mode 0600 where supported.

Mutation posture:
  HTTP command routes mutate only through their accepted command contracts.
  Existing observability and entity routes require Authorization bearer credentials, not Operator-Key.
  Operator-Key gates these operator instruction routes:
    /operations/instructions/pause
    /operations/instructions/release-expired-lease
    /operations/instructions/force-release-lease
    /operations/instructions/move-item
    /operations/instructions/move-entries
    /operations/instructions/move-available
    /operations/instructions/drop
    /operations/instructions/route-outstanding

Non-effects:
  serve does not generate, rotate, list, revoke, issue, enroll, or store keys.
  Rotation is verifier document replacement followed by restart.`
}

func credentialHelp() string {
	return `credential

commands:
  credential generate

Purpose:
  Creates bearer material for operators to place outside the substrate.

Non-effects:
  stores nothing, rotates nothing, lists nothing, revokes nothing, issues nothing, enrolls nothing.`
}

func credentialGenerateHelp() string {
	return `credential generate

Purpose:
  prints one raw key and one verifier_digest as JSON, then exits.

Mutation posture:
  Mutates no Registry vocabulary and writes no files.

Inputs:
  Reads operating system randomness only.

Non-effects:
  Stores nothing, rotates nothing, lists nothing, revokes nothing, issues nothing, enrolls nothing.
  The raw key is bearer authority.`
}

func initHelp() string {
	return `init

Purpose:
  writes a declaration file template.

Inputs:
  --from <path>  declaration path
  --yes          overwrite an existing file

Mutation posture:
  writes the declaration file path only.

Non-effects:
  does not read the database, network, credentials, verifier documents, or operator keys.
  does not mutate Registry vocabulary.`
}

func previewHelp() string {
	return `preview

Purpose:
  reads declaration and database inputs, then prints planned Registry vocabulary changes.

Inputs:
  --database-url <url>  database connection string
  --from <path>         declaration file
  --config <path>       process config file

Mutation posture:
  mutates no Registry vocabulary.

Non-effects:
  Does not run serve, generate credentials, rotate keys, or require Operator-Key.`
}

func applyHelp() string {
	return `apply

Purpose:
  reads declaration and database inputs, then applies Registry vocabulary changes.

Inputs:
  --database-url <url>  database connection string
  --from <path>         declaration file
  --config <path>       process config file

Mutation posture:
  mutates Registry vocabulary through the accepted declaration apply path.

Non-effects:
  Does not run serve, generate credentials, rotate keys, or require Operator-Key.`
}
