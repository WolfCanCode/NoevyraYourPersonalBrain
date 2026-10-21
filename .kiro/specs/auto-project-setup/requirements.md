# Requirements Document

## Introduction

The auto-project-setup feature gives Noevyra users zero-friction project onboarding. A single `brain setup` command detects the project's type from bounded repository metadata, generates a persistent Project_Profile stored in the local SQLite database, discovers best-practice skills and MCP servers for the detected project type, and provisions user-approved Capabilities with their configuration. The stored profile travels with the developer across every supported Agent (Claude, Codex, Cursor, and Kiro), eliminating manual re-setup when switching agents or resuming sessions.

This feature extends the existing `brain profile` detection into a full persistence and provisioning workflow, adds Kiro as a supported Agent in `brain run`, and integrates setup visibility into `brain doctor` and a new `brain setup --status` sub-command.

## Glossary

- **Agent**: One of the four supported coding agents: Claude, Codex, Cursor, or Kiro. Each Agent has a distinct CLI executable name used to launch it.
- **Auto_Setup**: The subsystem that orchestrates project detection, Capability discovery, user approval, and provisioning in a single Setup_Session.
- **Capability**: A named skill, MCP server, or CLI tool that the Auto_Setup can discover and provision for a project.
- **Capability_Catalog**: The registry of available Capabilities and the project-signal trigger rules used to match Capabilities to a Project_Profile.
- **CLI**: The `brain` command-line interface through which users invoke all Noevyra commands.
- **Context_File**: A temporary file created with file permissions `0600` that contains a compiled context pack and is passed to an Agent process by `brain run`.
- **Profile_Store**: The component within the local SQLite database (`brain.sqlite`) that persists and retrieves Project_Profiles.
- **Project_Profile**: A persisted record containing a project's detected languages, stacks, operational signals, manifests, file counts, provisioned Capabilities, and creation timestamp, keyed by the project identifier.
- **Project_Root**: The nearest ancestor directory containing a `.git` folder, determined by walking up from the invocation directory; falls back to the invocation directory when no `.git` folder is found within 10 ancestor levels.
- **Provisioner**: The Auto_Setup sub-component responsible for installing and configuring user-approved Capabilities.
- **Setup_Session**: A single invocation of `brain setup` that spans project detection, Capability discovery, user approval, and provisioning.

## Requirements

### Requirement 1: Auto-Setup Command and Project Detection

**User Story:** As a developer, I want to run a single `brain setup` command that detects my project type automatically, so that I do not need to manually describe my stack before receiving tailored capability recommendations.

#### Acceptance Criteria

1. WHEN `brain setup` is invoked in a directory, THE Auto_Setup SHALL identify the Project_Root by traversing ancestor directories upward from the invocation directory and returning the nearest ancestor directory that contains a `.git` folder, up to a maximum of 10 ancestor levels.
2. WHEN `brain setup` is invoked, THE Auto_Setup SHALL scan repository metadata bounded to 4000 files and derive the project's languages, stacks, and operational signals using the same detection rules as `brain profile`.
3. IF the Auto_Setup file scan reaches 4000 files before all subtrees are exhausted, THEN THE Auto_Setup SHALL stop scanning, set the `truncated` field to `true` in the resulting Project_Profile, and continue the Setup_Session with the partial data.
4. WHEN `brain setup` is invoked, THE Auto_Setup SHALL complete the project detection phase within 5 seconds on a repository of 4000 files or fewer stored on local disk.
5. WHEN `brain setup` is invoked, IF one or more of the following manifests is present at Project_Root — `go.mod`, `package.json`, `Cargo.toml`, `pyproject.toml`, `requirements.txt`, `pom.xml`, `build.gradle`, or `build.gradle.kts` — THEN THE Auto_Setup SHALL parse each detected manifest to extract dependency names and include them in the Project_Profile `stacks` field.
6. WHEN `brain setup` completes project detection, THE Auto_Setup SHALL write the detected languages, stacks, and operational signals to standard output before proceeding to Capability discovery.
7. IF no `.git` folder is found within 10 ancestor levels of the invocation directory, THEN THE Auto_Setup SHALL set Project_Root to the invocation directory.
8. IF a manifest file at Project_Root cannot be parsed, THEN THE Auto_Setup SHALL skip that manifest, record an error indication identifying the unparseable file by name, and continue processing any remaining manifests.

### Requirement 2: Persistent Project Profile Storage

**User Story:** As a developer, I want my project profile stored in the local brain database, so that switching agents or resuming a session does not require re-running detection.

#### Acceptance Criteria

1. WHEN a Setup_Session completes, THE Profile_Store SHALL persist the resulting Project_Profile in the local SQLite database at the canonical absolute path returned by `BRAIN_HOME` resolution, keyed by the deterministic project identifier derived from Project_Root.
2. THE Profile_Store SHALL persist Project_Profiles without requiring network connectivity.
3. WHEN a new Project_Profile would cause the total stored profiles for a project identifier to exceed three, THE Profile_Store SHALL delete the oldest-by-creation-timestamp profile for that project identifier before persisting the new profile.
4. WHEN `brain setup` is invoked for a Project_Root that already has a stored Project_Profile, THE Auto_Setup SHALL display the existing profile's project identifier, creation timestamp, and detected configuration summary, then require the user to confirm before overwriting it.
5. WHEN `brain setup --refresh` is invoked, THE Auto_Setup SHALL regenerate the Project_Profile by re-running detection without presenting the confirmation prompt required when an existing profile is present.
6. WHEN the Profile_Store serializes a Project_Profile to JSON and then deserializes the resulting JSON, THE Profile_Store SHALL produce a Project_Profile with fields and values identical to the original Project_Profile (round-trip property).
7. IF the user does not confirm when prompted by `brain setup` for an existing profile, THEN THE Auto_Setup SHALL abort the Setup_Session, make no changes to the database, and display a cancellation message.
8. IF the SQLite write fails during profile persistence, THEN THE Profile_Store SHALL abort the persist operation, preserve all existing profiles unchanged, and report an error message indicating the failure.

### Requirement 3: Capability Discovery

**User Story:** As a developer, I want Noevyra to discover best-practice skills, MCPs, and CLI tools matched to my detected project type, so that I receive a prioritized recommendation without manually searching for the right tools.

#### Acceptance Criteria

1. WHEN a Project_Profile is generated, THE Auto_Setup SHALL query the Capability_Catalog for all Capabilities where the Capability's trigger signals contain at least one value that matches, by case-insensitive exact comparison, any term present in the Project_Profile `languages`, `stacks`, or `signals` fields.
2. THE Capability_Catalog SHALL include at least one Capability entry for each of the following signal values: `go`, `rust`, `python`, `typescript`, `javascript`, `database`, `web`, `api`, `security`, `containers`, and `ci-cd`.
3. WHEN the Auto_Setup queries the Capability_Catalog and finds matching Capabilities, THE Auto_Setup SHALL rank them by each Capability's assigned confidence score in descending order and present at most 10 Capabilities to the user for review.
4. WHEN `brain setup --dry-run` is invoked, THE Auto_Setup SHALL present the ranked Capability list without modifying any file, configuration entry, or database record.
5. THE Auto_Setup SHALL label each discovered Capability as `local` when provisioning requires no network access, and as `network` when provisioning requires network access.
6. WHEN the Auto_Setup generates capability recommendations, THE Auto_Setup SHALL include any skills stored in the `approved_skills` table whose `keyword` value matches, by case-insensitive exact comparison, any term present in the Project_Profile `languages`, `stacks`, or `signals` fields.
7. IF the Capability_Catalog query returns no matching Capabilities, THEN THE Auto_Setup SHALL display a message indicating that no capabilities were matched for the detected project profile and proceed to the provisioning step with an empty selection.

### Requirement 4: Capability Provisioning with User Approval

**User Story:** As a developer, I want Noevyra to install and configure the capabilities I select in one step, so that my project is ready to work on without running separate tool-setup commands.

#### Acceptance Criteria

1. WHEN the user selects one or more Capabilities from the discovery list, THE Provisioner SHALL install and configure each selected Capability in the order presented.
2. WHEN the Provisioner provisions a skill Capability, THE Provisioner SHALL prompt the user for explicit approval for that specific skill before registering the skill's reference and keyword in the local skill trust store.
3. WHEN the Provisioner provisions an MCP Capability and at least one Agent is detected as installed on the system, THE Provisioner SHALL write a valid MCP server configuration entry to the configuration file for each detected installed Agent.
4. IF a Capability provisioning step produces an error, THEN THE Provisioner SHALL record the error message in the `events` table as a `setup_session` event, skip the failed Capability, and continue provisioning the remaining selected Capabilities.
5. WHEN all selected Capabilities have been processed, THE Provisioner SHALL display a provisioning summary that includes each Capability's name and its outcome classified as one of: `provisioned`, `skipped`, or `failed`.
6. WHEN a Capability that was previously provisioned is selected for provisioning, THE Provisioner SHALL update the existing configuration in place, maintaining exactly one entry per Capability in each configuration file and database table.
7. WHEN `brain setup --dry-run` is invoked, THE Provisioner SHALL display the planned provisioning steps without prompting for user approval and without writing to any file, configuration entry, or database record.
8. IF the user denies approval for a skill Capability during provisioning, THEN THE Provisioner SHALL classify that Capability's outcome as `skipped` and make no changes to the skill trust store.
9. IF the Provisioner attempts to write an MCP Capability configuration and no Agent is detected as installed on the system, THEN THE Provisioner SHALL classify that Capability's outcome as `failed` and record the failure in the `events` table.

### Requirement 5: Agent-Agnostic Profile Export

**User Story:** As a developer, I want my stored project profile included automatically when I launch any supported agent, so that I do not need to re-describe my project when switching between Claude, Codex, Cursor, and Kiro.

#### Acceptance Criteria

1. WHEN `brain run <agent> <task>` is invoked for a Project_Root that has a stored Project_Profile, THE Auto_Setup SHALL make the stored Project_Profile available to the Agent process by including it in the Context_File at the path provided to the Agent.
2. WHEN `brain run <agent> <task>` is invoked, IF no stored Project_Profile exists for the current Project_Root, THEN THE Auto_Setup SHALL derive a temporary profile using the detection rules defined in Requirement 1 and include it in the Context_File.
3. THE Auto_Setup SHALL support Context_File generation for the following Agent values: `claude`, `codex`, `cursor`, and `kiro`.
4. WHEN `brain run <agent>` is invoked, THE Auto_Setup SHALL create the Context_File at a temporary path within the directory returned by `BRAIN_HOME` resolution with file permissions `0600`.
5. WHEN the Agent process started by `brain run <agent>` exits, THE Auto_Setup SHALL delete the Context_File regardless of the Agent process exit code.
6. IF an unsupported value is passed as `<agent>`, THEN THE Auto_Setup SHALL write an error message to stderr listing the supported agent values and exit without creating a Context_File or starting any process.
7. IF Context_File creation fails, THEN THE Auto_Setup SHALL write an error message to stderr and exit without starting the Agent process.

### Requirement 6: Kiro Agent Support

**User Story:** As a developer, I want to launch Kiro with the same brain context as Claude, Codex, and Cursor, so that I have a consistent, context-aware startup experience across all supported agents.

#### Acceptance Criteria

1. THE CLI SHALL accept `kiro` as a valid value for `<agent>` in the `brain run <agent> <task>` command without producing an unrecognized-agent error.
2. WHEN `brain run kiro <task>` is invoked, THE CLI SHALL compile a context pack for the current project and write it to a Context_File in a newly created temporary session directory under the brain data directory, with file permissions `0600`, before starting the Kiro CLI process.
3. WHEN `brain run kiro <task>` is invoked, THE CLI SHALL pass the absolute path of the Context_File to the Kiro CLI process via the `BRAIN_CONTEXT_FILE` environment variable so the Kiro session has access to the project context.
4. IF context pack compilation or Context_File write fails when `brain run kiro <task>` is invoked, THEN THE CLI SHALL write an error message to stderr and exit without starting the Kiro CLI process.
5. WHEN the Kiro CLI process started by `brain run kiro` exits, THE CLI SHALL delete the temporary session directory and all its contents regardless of the Kiro CLI process exit code.
6. WHEN `brain mcp-config` is invoked, THE CLI SHALL include a Kiro-compatible MCP server configuration entry as a key under `mcpServers`, with a `command` field set to the absolute path of the `brain` binary and an `args` field set to `["mcp"]`.
7. IF the `kiro` executable is not found in the system PATH when `brain run kiro` is invoked, THEN THE CLI SHALL write an error message to stderr that names the missing executable as `kiro` and instructs the user to install the Kiro CLI before retrying.
8. WHEN `brain doctor` is invoked, THE CLI SHALL include a `kiro` key in the `commands` map with a boolean value indicating whether the `kiro` executable was found in the system PATH.

### Requirement 7: Setup Visibility and Diagnostics

**User Story:** As a developer, I want to inspect the current setup status for my project, so that I know what capabilities have been provisioned and whether my profile reflects the current project state.

#### Acceptance Criteria

1. WHEN `brain setup --status` is invoked, THE Auto_Setup SHALL display the stored Project_Profile for the current Project_Root including: detected languages, stacks, signals, provisioned Capabilities, and profile creation timestamp.
2. IF `brain setup --status` is invoked and no stored Project_Profile exists for the current Project_Root, THEN THE Auto_Setup SHALL display a message indicating that no profile exists for the current project and instructing the user to run `brain setup`.
3. WHEN `brain doctor` is invoked, THE CLI SHALL report, for the current Project_Root: whether a stored Project_Profile exists, and for each Capability listed in the stored Project_Profile as provisioned, whether that Capability is present in the local MCP configuration files.
4. WHEN a Setup_Session terminates, THE Auto_Setup SHALL record a `setup_session` event in the `events` table containing: the project identifier, the names of all Capabilities that were attempted during the session, and the session outcome classified as `completed` if all attempted Capabilities were successfully provisioned, `partial` if at least one but not all attempted Capabilities were successfully provisioned, or `failed` if no attempted Capabilities were successfully provisioned.
5. IF `brain doctor` is invoked and the local MCP configuration files cannot be read, THEN THE CLI SHALL report that MCP configuration files are inaccessible and list the file paths that were checked.
