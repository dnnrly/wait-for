Feature: Configuration from CLI

  Scenario: Usage exits with error
    When I run wait-for with parameters "-h"
    Then the output contains "Usage of"
    And wait-for exits with an error

  Scenario: Waits on a HTTP service
    Given I have an HTTP server running on port 80 that responds with 200
    Given I have an HTTP server running on port 8080 that responds with 200
    When I run wait-for with parameters "http://localhost/health"
    Then I can see that an HTTP request was made for "localhost GET /health"
    And the output contains "started waiting for http://localhost/health"
    And the output contains "finished waiting for http://localhost/health"
    And wait-for exits without error

  Scenario: Waits on a TCP connection
    Given I have an HTTP server running on port 80 that waits "10s" then responds with 200
    Given I have an HTTP server running on port 8080 that responds with 200
    When I run wait-for with parameters "tcp:localhost:80"
    Then I can see a connection event "127.0.0.1:80 new"
    And the output contains "started waiting for tcp:localhost:80"
    And the output contains "finished waiting for tcp:localhost:80"
    And wait-for exits without error
    And the time taken is less than "10s"

  Scenario: Waits on several services
    Given I have an HTTP server running on port 80 that responds with 200
    When I run wait-for with parameters "http://localhost/health http://localhost/another"
    Then I can see that an HTTP request was made for "localhost GET /health"
    And I can see that an HTTP request was made for "localhost GET /another"
    And the output contains "finished waiting for http://localhost/health"
    And the output contains "finished waiting for http://localhost/another"
    And wait-for exits without error

  Scenario: Fails when HTTP listener response with 500 error
    Given I have an HTTP server running on port 80 that responds with 500
    When I run wait-for with parameters "http://localhost/health"
    Then I can see that an HTTP request was made for "localhost GET /health"
    And the output contains "error while waiting for http://localhost/health"
    And the output contains "timed out waiting for http://localhost/health"
    And wait-for exits with an error

  Scenario: Fails when HTTP listener response with 400 error
    Given I have an HTTP server running on port 80 that responds with 400
    When I run wait-for with parameters "http://localhost/health"
    Then I can see that an HTTP request was made for "localhost GET /health"
    And wait-for exits with an error

  Scenario: Times out if it can't connect to a service
    When I run wait-for with parameters "http://non-existent/health"
    Then wait-for exits with an error
    And the time taken is more than "5s"
    And the output contains "timed out waiting for http://non-existent/health"

  Scenario: Times out if HTTP service hangs
    Given I have an HTTP server running on port 80 that waits "10s" then responds with 200
    When I run wait-for with parameters "http://localhost/health"
    Then wait-for exits with an error
    And I can see that an HTTP request was made for "localhost GET /health"
    And the time taken is more than "5s"

  Scenario: Waits until a service comes up
    Given I have an HTTP server running on port 80 that responds with 500 for "3s" then responds with 200
    When I run wait-for with parameters "http://localhost/health"
    Then wait-for exits without error
    And the output contains "error while waiting for http://localhost/health"
    And the output contains "finished waiting for http://localhost/health"
    And the time taken is more than "3s"
    And the time taken is less than "5s"

  Scenario: Timeout is configurable
    When I run wait-for with parameters "-timeout 2s http://non-existent/health"
    Then wait-for exits with an error
    And the time taken is more than "2s"
    And the time taken is less than "5s"
    And the output contains "timed out waiting for http://non-existent/health"

  Scenario: Quiet option removes output when successful
    Given I have an HTTP server running on port 80 that responds with 500 for "3s" then responds with 200
    When I run wait-for with parameters "-quiet http://localhost/health"
    Then wait-for exits without error
    And the output does not contain "error while waiting for http://localhost/health"
    And the output does not contain "timed out waiting for http://localhost/health"
    And the output does not contain "finished waiting for http://localhost/health"

  Scenario: Quiet option removes output when failed
    Given I have an HTTP server running on port 80 that responds with 500
    When I run wait-for with parameters "-quiet http://localhost/health"
    Then wait-for exits with an error
    And the output does not contain "Usage of"
    And the output does not contain "error while waiting for http://localhost/health"
    And the output does not contain "finished waiting for http://localhost/health"
    And the output contains "timed out waiting for http://localhost/health"
      # We still want to know what failed!

