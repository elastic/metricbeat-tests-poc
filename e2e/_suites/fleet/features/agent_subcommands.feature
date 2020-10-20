@agent_subcommands
Feature: Agent subcommands
  Scenarios for the Agent to test the various subcommands connecting it to Fleet.

@install-agent
Scenario Outline: Deploying the <os> agent with install command
  When a "<os>" agent is deployed to Fleet with with "tar" installer
  Then the "elastic-agent" process is in the "started" state on the host
    And the "filebeat" process is in the "started" state on the host
    And the "metricbeat" process is in the "started" state on the host
    And the agent is listed in Fleet as "online"
    And system package dashboards are listed in Fleet
Examples:
| os     |
| centos |
| debian |

@stop-agent
Scenario Outline: Stopping an 'installed' <os> agent stops backend processes
  Given a "<os>" agent is deployed to Fleet with "tar" installer
  When the "elastic-agent" process is "stopped" on the host
  Then the "filebeat" process is in the "stopped" state on the host
    And the "metricbeat" process is in the "stopped" state on the host
Examples:
| os     |
| centos |
| debian |

@restart-host
Scenario Outline: Restarting the installed <os> host restarts backend processes
  Given a "<os>" agent is deployed to Fleet with "tar" installer
  When the host is restarted
  Then the "elastic-agent" process is in the "started" state on the host
    And the "filebeat" process is in the "started" state on the host
    And the "metricbeat" process is in the "started" state on the host
Examples:
| os     |
| centos |
| debian |

@unenroll-host
Scenario Outline: Un-enrolling the installed <os> agent
  Given a "<os>" agent is deployed to Fleet with "tar" installer
  When the agent is un-enrolled
  Then the "elastic-agent" process is in the "started" state on the host
    And the agent is listed in Fleet as "inactive"
    And the "filebeat" process is in the "stopped" state on the host
    And the "metricbeat" process is in the "stopped" state on the host
Examples:
| os     |
| centos |
| debian |

@reenroll-host
Scenario Outline: Re-enrolling the installed <os> agent
  Given a "<os>" agent is deployed to Fleet with "tar" installer
    And the agent is un-enrolled
    And the "elastic-agent" process is "stopped" on the host
  When the agent is re-enrolled on the host
    And the "elastic-agent" process is "started" on the host
  Then the agent is listed in Fleet as "online"
Examples:
| os     |
| centos |
| debian |

@uninstall-host
Scenario Outline: Un-installing the installed <os> agent
  Given a "<os>" agent is deployed to Fleet with "tar" installer
  When the "elastic-agent" process is "uninstalled" on the host
  Then the "elastic-agent" process is in the "stopped" state on the host
    And the agent is listed in Fleet as "inactive"
    And the "filebeat" process is in the "stopped" state on the host
    And the "metricbeat" process is in the "stopped" state on the host
    And the file system Agent folder is empty
Examples:
| os     |
| centos |
| debian |

@restart-agent
Scenario Outline: Restarting the installed <os> agent
  Given a "<os>" agent is deployed to Fleet with "tar" installer
  When the "elastic-agent" process is "restarted" on the host
    And the agent is listed in Fleet as "online"
    And the "filebeat" process is in the "started" state on the host
    And the "metricbeat" process is in the "started" state on the host
Examples:
| os     |
| centos |
| debian |
