Feature: Nudge for review on open pull requests
  As a developer who raises pull requests
  I want a reminder sent for each pull request of mine that still needs review
  So that a pull request is never left waiting for review without me noticing

  Background:
    Given I am the pull request author "john.smith"

  Rule: Only my pull requests that are still awaiting review are nudged

    Scenario: A pull request is nudged while it is open, ready for review, and unapproved
      Given I have an open, non-draft, unapproved pull request titled "Add retry logic to worker"
      When the nudge tool runs
      Then a nudge is sent for "Add retry logic to worker"

    Scenario: A pull request that has already received an approval is not nudged
      Given I have an open pull request titled "Fix flaky test" that has already been approved
      When the nudge tool runs
      Then no nudge is sent for "Fix flaky test"

    Scenario: A draft pull request is not nudged
      Given I have an open draft pull request titled "Draft: new feature"
      When the nudge tool runs
      Then no nudge is sent for "Draft: new feature"

    Scenario: A pull request that is put back into draft is no longer nudged
      Given I have an open, non-draft, unapproved pull request titled "Add retry logic to worker"
      And it is put back into draft
      When the nudge tool runs
      Then no nudge is sent for "Add retry logic to worker"

    Scenario: A pull request I did not author is not nudged
      Given "someone.else" has an open pull request titled "Refactor auth module"
      When the nudge tool runs
      Then no nudge is sent for "Refactor auth module"

    Scenario: A merged or closed pull request is not nudged
      Given I have a merged pull request titled "Old change, already merged"
      And I have a closed pull request titled "Abandoned change"
      When the nudge tool runs
      Then no nudge is sent for "Old change, already merged"
      And no nudge is sent for "Abandoned change"

    Scenario: Every pull request still awaiting review gets its own nudge
      Given I have the following pull requests awaiting review:
        | title                      |
        | Add retry logic to worker  |
        | Fix flaky test             |
      When the nudge tool runs
      Then a nudge is sent for "Add retry logic to worker"
      And a nudge is sent for "Fix flaky test"

    Scenario: No nudges are sent when nothing is awaiting review
      Given I have no pull requests awaiting review
      When the nudge tool runs
      Then no nudge is sent

  Rule: The reminder reflects whether this is the first nudge or a follow-up

    Scenario: The first nudge for a pull request asks for a review
      Given I have a pull request titled "Add retry logic to worker" awaiting review
      And it has not been nudged before
      When the nudge tool runs
      Then the nudge for "Add retry logic to worker" asks for a review

    Scenario: A follow-up nudge notes the review is still outstanding
      Given I have a pull request titled "Add retry logic to worker" awaiting review
      And it was nudged on a previous run
      When the nudge tool runs
      Then the nudge for "Add retry logic to worker" notes that review is still outstanding

    Scenario: The reminder stops escalating once it has nudged this pull request many times
      Given I have a pull request titled "Add retry logic to worker" awaiting review
      And it has already been nudged 10 times
      When the nudge tool runs
      Then the nudge for "Add retry logic to worker" is the final reminder message

  Rule: A nudge always identifies the pull request

    Scenario: The nudge names the pull request and links to it
      Given I have a pull request titled "Add retry logic to worker" at "https://git.example.com/group/project-a/-/merge_requests/42" awaiting review
      When the nudge tool runs
      Then the nudge for "Add retry logic to worker" includes its title and a link to "https://git.example.com/group/project-a/-/merge_requests/42"

  Rule: Nudge history is kept up to date, and only for pull requests still awaiting review

    Scenario: A pull request is remembered as nudged once it has been nudged
      Given I have a pull request titled "Add retry logic to worker" awaiting review
      And it has not been nudged before
      When the nudge tool runs
      Then "Add retry logic to worker" is remembered as nudged

    Scenario: History is forgotten once a pull request is no longer awaiting review
      Given "Old change, already merged" was nudged before
      And it is now merged
      When the nudge tool runs
      Then "Old change, already merged" is no longer remembered

    Scenario: A pull request that returns from draft is nudged as a fresh review request
      Given I had a pull request titled "Add retry logic to worker" that was nudged 2 times
      And it was put back into draft and then marked ready for review again
      When the nudge tool runs
      Then the nudge for "Add retry logic to worker" asks for a review
