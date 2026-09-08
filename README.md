[//]: # (STANDARD README)
[//]: # (https://github.com/RichardLitt/standard-readme)
[//]: # (----------------------------------------------)
[//]: # (Uncomment optional sections as required)
[//]: # (----------------------------------------------)
[//]: # (AI INSTRUCTIONS)
[//]: # (Hello friendly AI agent! Please do your best to comply with)
[//]: # (the following style guide in this document.)
[//]: # (> Comments)
[//]: # (All [//]: # comment lines in this file are part of)
[//]: # (the Standard README template structure. Preserve them exactly!)
[//]: # (Do not remove, reword, merge, or move them when editing content.)
[//]: # (This includes comments inside sections that already have content.)
[//]: # (> Line length)
[//]: # (Keep all prose lines to a maximum of 88 characters. Exclude tables)
[//]: # (and fenced code blocks from this limit.)
[//]: # (----------------------------------------------)



[//]: # (Title)
[//]: # (Match repository name)
[//]: # (REQUIRED)

# reviewreminder

[//]: # (Banner)
[//]: # (OPTIONAL)
[//]: # (Must not have its own title)
[//]: # (Must link to local image in current repository)



[//]: # (Badges)
[//]: # (OPTIONAL)
[//]: # (Must not have its own title)



[//]: # (Short description)
[//]: # (REQUIRED)
[//]: # (An overview of the intentions of this repo)
[//]: # (Must not have its own title)
[//]: # (Must be less than 120 characters)
[//]: # (Must match GitHub's description)

Remind colleagues to review your code

[//]: # (Long Description)
[//]: # (OPTIONAL)
[//]: # (Must not have its own title)
[//]: # (A detailed description of the repo)

Many teams review code by posting a link to a pull request in Slack and hoping
someone picks it up. Those messages are easy to ignore, so a pull request can sit
unreviewed for days while its author naturally moves on to other, higher-priority
work, and chasing reviews by hand is suboptimal.

`reviewreminder` automates the chasing. It finds your open pull requests that are
still awaiting review and posts a nudge for each one to a webhook of your choosing,
until the review lands, so you never have to remember to follow up. Assigning
reviewers with your forge's built-in feature would be tidier, but most people cannot
change how their team works: if you can't beat the process, automate it.

## Table of Contents

[//]: # (REQUIRED)
[//]: # (Managed automatically)
[//]: # (Changes between TABLE_OF_CONTENTS_START and TABLE_OF_CONTENTS_END)
[//]: # (markers will be overwritten)
[//]: # (Set `tocgen: true` in estate-repos/repos.yaml)
[//]: # (to enable automatic table of contents management.)
[//]: # (Or just do it manually like a weirdo, you do you...)

[//]: # (TOCGEN_TABLE_OF_CONTENTS_START)

- [Install](#install)
- [Usage](#usage)
- [Documentation](#documentation)
- [Repository Configuration](#repository-configuration)
- [Contributing](#contributing)
- [License](#license)
    - [Code](#code)
    - [Non-code content](#non-code-content)

[//]: # (TOCGEN_TABLE_OF_CONTENTS_END)

[//]: # (## Security)
[//]: # (OPTIONAL)
[//]: # (May go here if it is important to highlight security concerns.)



[//]: # (## Background)
[//]: # (OPTIONAL)
[//]: # (Explain the motivation and abstract dependencies for this repo)



## Install

[//]: # (Explain how to install the thing.)
[//]: # (OPTIONAL IF documentation repo)
[//]: # (ELSE REQUIRED)

`reviewreminder` is distributed through our Homebrew tap. It installs the `rr`
command:

```bash
brew install evoteum/homebrew-tap/reviewreminder
```

## Usage
[//]: # (REQUIRED)
[//]: # (Explain what the thing does. Use screenshots and/or videos.)

Run `rr init` once to create your configuration:

```bash
rr init
```

This creates `~/.reviewreminder/`, holding everything the tool needs:

```
~/.reviewreminder/
├── config.yaml     # forge, webhook URL, nudge interval, and an optional schedule
├── messages.yaml   # the reminder messages, from polite to insistent
├── history.json    # which pull requests have been nudged, and when (managed by rr)
└── token           # empty, chmod 600 — put your forge access token here
```

Example `config.yaml` and `messages.yaml` files are in [`examples`](examples) for
reference. Fill in your forge details and `webhook_url` in `config.yaml`, and put
your forge access token in the `token` file. Keeping the token in its own file means
`config.yaml` holds no secrets, so it is safe to template with a tool like Ansible,
while the token can be placed on the machine out of band and never has to pass
through any automation. Then run `rr` to check your open pull requests and send a
nudge for each one that is still awaiting review:

```bash
rr
```

For each nudge, `rr` sends an HTTP POST to your webhook with three fields:

| Field      | Description                             |
| ---------- | --------------------------------------- |
| `message`  | The nudge text (see escalation below)   |
| `pr_title` | The title of the pull request           |
| `pr_url`   | A link to the pull request              |

What happens next is up to the webhook. If your team uses Slack, point `rr` at a
[Slack workflow](https://slack.com/help/articles/360053571454) webhook that takes
these fields and posts a message to your team's PR review request channel — so your
nudges land wherever reviews are already requested, without `rr` needing to know
anything about Slack.

The messages in `messages.yaml` escalate: the first nudge for a pull request asks
politely for a review, and each follow-up gets more insistent until the last message
is reached and the tone stops escalating. Define as many or as few as you like. A
pull request stops being nudged as soon as it is approved, merged, closed, or put
back into draft, and only your own pull requests are nudged.

### GitLab

`rr` currently supports GitLab. It reads the open, non-draft, unapproved merge
requests you authored, using the `read_api` scope of the token in your `token` file.

It considers your 100 most recent open merge requests. If you genuinely have more
than 100 open at once awaiting review, your team is slower at reviewing than this
tool can paper over; please
[log an issue](https://github.com/evoteum/reviewreminder/issues) and we'll add
pagination.

### Nudge frequency

`rr` records what it has nudged, and when, in `history.json`, so how often it runs is
separate from how often any one pull request is nudged. A newly discovered pull
request is nudged straight away; after that, `rr` waits at least `nudge_interval`
before nudging the same one again:

```yaml
# config.yaml
nudge_interval: "24h"   # nudge each pull request at most once a day
```

That means you can run `rr` frequently — so new pull requests are picked up and sent
on promptly — without over-nudging: a run every 15 minutes with `nudge_interval:
"24h"` still nudges each pull request only once a day.

`nudge_interval` can only space nudges out, never speed them up. Each pull request is
nudged at most once per run, so the real cadence is whichever is longer — your
schedule or `nudge_interval`. If `rr` runs only once a day, a `nudge_interval` of
`4h` still results in one nudge a day. Set `nudge_interval` no shorter than how often
you run `rr`, or it has no effect.

### Scheduling

Scheduling is off by default — `rr` only runs when you run it. To have it run
automatically, enable scheduling in `config.yaml` with a cron expression and run `rr`
once to install the schedule:

```yaml
# config.yaml
schedule:
  enabled: true
  expression: "*/15 * * * *"   # check every 15 minutes
```

The configuration file is the single source of truth: change the expression and the
next run updates the schedule; set `enabled: false` and the next run removes it. The
expression is kept while disabled, so you can pause and resume without retyping it,
and there is nothing to export in your shell.

The cron expression is interpreted in the machine's local timezone, like any other
crontab entry, so `0 9 * * 1-5` means 9am local time and follows daylight-saving
changes. The nudge interval, by contrast, is an absolute duration, so it is
unaffected by timezone or daylight saving.

`rr` schedules itself through your user crontab. On macOS this runs without extra
setup for the default log location, but if cron jobs do not fire on your machine,
grant Full Disk Access to `/usr/sbin/cron` in System Settings › Privacy & Security.

[//]: # (Extra sections)
[//]: # (OPTIONAL)
[//]: # (This should not be called "Extra Sections".)
[//]: # (This is a space for ≥0 sections to be included,)
[//]: # (each of which must have their own titles.)



## Documentation

Further documentation is in the [`docs`](docs/) directory.

## Repository Configuration

> [!WARNING]
> This repo is controlled by OpenTofu in the [estate-repos](https://github.com/evoteum/estate-repos) repository.
>
> Manual configuration changes will be overwritten the next time OpenTofu runs.

[//]: # (## Requirements)
[//]: # (OPTIONAL)
[//]: # (List runtime and toolchain prerequisites with minimum versions.)



[//]: # (## API)
[//]: # (OPTIONAL)
[//]: # (Describe exported functions and objects)



[//]: # (## Maintainers)
[//]: # (OPTIONAL)
[//]: # (List maintainers for this repository)
[//]: # (along with one way of contacting them - GitHub link or email.)



[//]: # (## Thanks)
[//]: # (OPTIONAL)
[//]: # (State anyone or anything that significantly)
[//]: # (helped with the development of this project)



## Contributing
[//]: # (REQUIRED)
If you need any help, please log an issue and one of our team will get back to you.

PRs are welcome.


## License
[//]: # (REQUIRED)

### Code

All source code in this repository is licenced under the GNU Affero General Public
License v3.0
[AGPL-3.0](https://www.gnu.org/licenses/agpl-3.0.en.html).
A copy of this is provided in the [LICENSE](LICENSE).

### Non-code content

All non-code content in this repository, including but not limited to images, diagrams
or prose documentation, is licenced under the Creative Commons Attribution-ShareAlike
4.0 International licence,
[CC BY-SA 4.0](https://creativecommons.org/licenses/by-sa/4.0/).
