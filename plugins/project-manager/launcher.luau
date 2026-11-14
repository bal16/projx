--!nonstrict

-- Project Manager launcher provider for Noctalia, backed by the pjx CLI.
-- Supports: opening projects, adding new projects from a path, tag filter
-- (#tag), and quick configuration actions (scan, import, set default editor).

local common = require("./lib/common.luau")
local results = require("./lib/results.luau")

local SCRIPT_VERSION = "v8-refactor"
noctalia.log("pjx launcher loaded: " .. SCRIPT_VERSION)

local cachedProjects = nil
local loadError = nil

local function loadProjects(onDone)
    noctalia.runAsync("pjx list --format=delimited 2>&1", function(result)
        local projects = {}
        if result.exitCode == 0 and type(result.stdout) == "string" then
            projects = common.parseProjects(result.stdout)
        end
        cachedProjects = projects
        loadError = (result.exitCode ~= 0) and common.trim(result.stderr or result.stdout or "") or nil
        noctalia.log(string.format("pjx: list exit=%d projects=%d", result.exitCode, #projects))
        onDone(projects)
    end)
end

function onQuery(query)
    local filter = common.trim(query)
    noctalia.log("pjx: onQuery '" .. filter .. "'")

    if filter == "" then
        cachedProjects = nil
    end

    if cachedProjects then
        results.showResults(query, cachedProjects, filter, loadError)
        return
    end

    launcher.setResults(query, {
        {
            id = "",
            title = noctalia.tr("loading") or "Loading projects...",
            subtitle = noctalia.tr("loading-subtitle") or "querying pjx",
            glyph = "loader",
        },
    })

    loadProjects(function(projects)
        results.showResults(query, projects, filter, loadError)
    end)
end

function onActivate(id)
    noctalia.log("pjx: onActivate '" .. id .. "'")
    if id == "" then
        return
    end

    if id:match("^add:") then
        local path = id:sub(5)
        noctalia.runAsync("pjx add " .. common.shellQuote(path) .. " 2>&1", function(result)
            if result.exitCode ~= 0 then
                noctalia.notify("projx", "Failed to add project: " .. (result.stderr or ""))
            else
                noctalia.notify("projx", "Project added: " .. path)
                cachedProjects = nil
            end
        end)
        return
    end

    if id:match("^vscprofile:") then
        local action = id:sub(12)
        if action:sub(1, 7) == "create:" then
            local name = action:sub(8)
            noctalia.runAsync("pjx profiles create " .. common.shellQuote(name) .. " 2>&1", function(result)
                if result.exitCode ~= 0 then
                    noctalia.notify("projx", "Failed: " .. (result.stderr or ""))
                else
                    noctalia.notify("projx", "Profile created: " .. name)
                end
            end)
        elseif action == "createprompt" then
            noctalia.notify("projx", "Type: profile create <name>")
        end
        return
    end

    if id:match("^util:") then
        local action = id:sub(6)
        noctalia.log("pjx: util action '" .. action .. "'")
        if action == "scan" then
            noctalia.runAsync("pjx scan 2>&1", function(_)
                noctalia.notify("projx", "Scan complete")
                cachedProjects = nil
            end)
        elseif action == "import-zed" then
            noctalia.runAsync("pjx import zed 2>&1", function(_)
                noctalia.notify("projx", "Imported from Zed")
                cachedProjects = nil
            end)
        elseif action == "import-all" then
            noctalia.runAsync("pjx import --all 2>&1", function(_)
                noctalia.notify("projx", "Import complete")
                cachedProjects = nil
            end)
        elseif action == "import-project-manager" then
            noctalia.runAsync("pjx import project-manager 2>&1", function(_)
                noctalia.notify("projx", "Imported from Project Manager")
                cachedProjects = nil
            end)
        elseif action == "refresh" then
            cachedProjects = nil
            noctalia.notify("projx", "Refreshed")
        elseif action == "profiles" then
            noctalia.runAsync("pjx profiles 2>&1", function(result)
                local out = common.trim(result.stdout or "")
                if out == "" then
                    out = common.trim(result.stderr or "")
                end
                if out == "" then
                    out = "no profiles found"
                else
                    out = out:gsub("\n", ", ")
                end
                noctalia.notify("VS Code profiles", out:sub(1, 200))
            end)
        elseif action:sub(1, 7) == "editor-" then
            local ed = action:sub(8)
            noctalia.runAsync("pjx config set default_editor " .. ed .. " 2>&1", function(_)
                noctalia.notify("projx", "Default editor set to " .. ed)
            end)
        end
        return
    end

    -- normal project open
    local newWindow = noctalia.getConfig("new_window") == true
    local args = "pjx open " .. common.shellQuote(id)
    if newWindow then
        args = args .. " --new-window"
    end
    noctalia.runAsync(args .. " 2>&1", function(result)
        noctalia.log(string.format("pjx: open exit=%d", result.exitCode))
        if result.exitCode ~= 0 then
            noctalia.notifyError(
                "Failed to open project",
                common.trim(result.stderr or result.stdout or "unknown error"):sub(1, 160)
            )
        end
    end)
end
