Feature: Configuration from files

  Scenario: Can wait on a named HTTP service in a config file
    Given I have an HTTP server running on port 80 that responds with 200
    When I run wait-for with parameters "-config fixtures/wait-for.yaml http-connection"
    Then I can see that an HTTP request was made for "localhost GET /health"
    And wait-for exits without error
    And the output contains "started waiting for http-connection"
    And the output contains "finished waiting for http-connection"

  Scenario: Waits on a TCP connection
    Given I have an HTTP server running on port 80 that waits "10s" then responds with 200
    When I run wait-for with parameters "-config fixtures/wait-for.yaml tcp-connection"
    Then I can see a connection event "127.0.0.1:80 new"
    And wait-for exits without error
    And the time taken is less than "2s"
    And the output contains "started waiting for tcp-connection"
    And the output contains "finished waiting for tcp-connection"

  Scenario: Times out if HTTP service takes longer the configuration allows
    Given I have an HTTP server running on port 81 that waits "10s" then responds with 200
    When I run wait-for with parameters "-config fixtures/wait-for.yaml timeout-connection"
    Then wait-for exits with an error
    And I can see that an HTTP request was made for "localhost:81 GET /health"
    And the time taken is less than "5s"

  Scenario: Setting default timeout does not override target timeout
    Given I have an HTTP server running on port 81 that waits "10s" then responds with 200
    When I run wait-for with parameters "-timeout 60s -config fixtures/wait-for.yaml timeout-connection"
    Then wait-for exits with an error
    And I can see that an HTTP request was made for "localhost:81 GET /health"
    And the time taken is less than "5s"

