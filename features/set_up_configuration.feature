Feature: Set up reviewreminder configuration
  As a developer installing reviewreminder
  I want a command that creates my configuration files
  So that I have a starting point to edit rather than writing them from scratch

  Rule: Initialising creates the configuration when none exists

    Scenario: Initialising for the first time
      Given no reviewreminder configuration exists
      When I initialise the configuration
      Then a configuration file is created for me to edit
      And a messages file is created for me to edit
      And an empty, private token file is created for me to add my access token
      And scheduling is disabled

  Rule: Initialising never overwrites an existing configuration

    Scenario: Initialising when configuration already exists
      Given a reviewreminder configuration already exists
      When I initialise the configuration
      Then my existing configuration is left unchanged
