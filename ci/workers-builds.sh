#!/bin/sh
# Workers Builds' old trigger command, until `mise run charter:builds-configure` points it at charter's launcher: the
# launcher itself (.mise/tasks/charter-builds, written by go tool charter tasks). Delete this file after that.
exec sh .mise/tasks/charter-builds "$@"
