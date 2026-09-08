Feature: Limit how often each pull request is nudged
  As a developer who raises pull requests
  I want to run the nudge tool often but cap how often any one pull request is nudged
  So that new pull requests are picked up promptly without the same one being over-nudged

  Background:
    Given I am the pull request author "john.smith"
    And the nudge interval is "24h"

  Rule: A newly discovered pull request is nudged straight away

    Scenario: A pull request that has never been nudged is nudged on the next run
      Given I have a pull request awaiting review that has never been nudged
      When the nudge tool runs
      Then a nudge is sent for that pull request

  Rule: A pull request is not nudged again until the interval has passed

    Scenario: A pull request nudged within the interval is left alone
      Given I have a pull request awaiting review that was last nudged 1 hour ago
      When the nudge tool runs
      Then no nudge is sent for that pull request

    Scenario: A pull request is nudged again once the interval has passed
      Given I have a pull request awaiting review that was last nudged 25 hours ago
      When the nudge tool runs
      Then a nudge is sent for that pull request
