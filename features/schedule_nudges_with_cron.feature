Feature: Schedule review nudges with a cron expression
  As a developer who raises pull requests
  I want to enable automatic runs with a cron expression in my configuration
  So that reminders are sent on a schedule I choose, and never without my say-so

  Rule: Scheduling is off unless it is enabled

    Scenario: Nudges are not scheduled when scheduling is disabled
      Given scheduling is disabled in the configuration
      And nudges are not currently scheduled
      When the nudge tool runs
      Then nudges are not scheduled

  Rule: When enabled, the schedule mirrors the configured expression

    Scenario: A schedule is created when scheduling is enabled
      Given scheduling is enabled with the expression "0 9 * * 1-5"
      And nudges are not currently scheduled
      When the nudge tool runs
      Then nudges are scheduled to run on "0 9 * * 1-5"

    Scenario: The schedule is updated when the expression changes
      Given nudges are scheduled to run on "0 9 * * 1-5"
      And the schedule expression is changed to "0 8,14 * * *"
      When the nudge tool runs
      Then nudges are scheduled to run on "0 8,14 * * *"

    Scenario: The schedule is removed when scheduling is disabled
      Given nudges are scheduled to run on "0 9 * * 1-5"
      And scheduling is disabled in the configuration
      When the nudge tool runs
      Then nudges are not scheduled

    Scenario: A schedule that already matches the configuration is left in place
      Given scheduling is enabled with the expression "0 9 * * 1-5"
      And nudges are scheduled to run on "0 9 * * 1-5"
      When the nudge tool runs
      Then nudges are scheduled to run on "0 9 * * 1-5"

    Scenario Outline: An invalid expression is refused and any schedule is left in place
      Given nudges are scheduled to run on "0 9 * * 1-5"
      And scheduling is enabled with the expression "<expression>"
      When the nudge tool runs
      Then I am told that "<expression>" is not a valid cron expression
      And nudges are scheduled to run on "0 9 * * 1-5"

      Examples:
        | expression |
        | not a cron |
        | 0 9 * *    |
        | 99 9 * * * |

  Rule: Enabling scheduling without an expression is rejected

    Scenario: Scheduling is enabled but no expression is set
      Given scheduling is enabled with no expression
      And nudges are not currently scheduled
      When the nudge tool runs
      Then I am told that a schedule expression is required
      And nudges are not scheduled
