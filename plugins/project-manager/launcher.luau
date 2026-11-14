--!nonstrict

-- Project Manager launcher provider for Noctalia, backed by the pjx CLI.
-- Supports: opening projects, adding new projects from a path, tag filter
-- (#tag), and quick configuration actions (scan, import, set default editor).

local SCRIPT_VERSION = "v7-util-prefix-fix"
noctalia.log("pjx launcher loaded: " .. SCRIPT_VERSION)
local cachedProjects = nil
local loadError = nil
local FIELD_SEP = "\31"

local function trim(s)
    return (s or ""):match("^%s*(.-)%s*$")
end

local function shellQuote(s)
    return "'" .. tostring(s):gsub("'", "'\"'\"'") .. "'"
end

local function splitFields(line)
    local fields = {}
    local start = 1
    while true do
        local sepStart, sepEnd = line:find(FIELD_SEP, start, true)
        if not sepStart then
            table.insert(fields, line:sub(start))
            break
        end
        table.insert(fields, line:sub(start, sepStart - 1))
        start = sepEnd + 1
    end
    return fields
end

local function loadProjects(onDone)
    local cmd = "pjx list --format=delimited 2>&1"
    noctalia.runAsync(cmd, function(result)
        local projects = {}
        if result.exitCode == 0 and type(result.stdout) == "string" then
            for line in result.stdout:gmatch("[^\n]+") do
                local f = splitFields(line)
                local id = trim(f[1])
                if id ~= "" then
                    table.insert(projects, {
                        id = id,
                        name = trim(f[2]),
                        path = trim(f[3]),
                        editor = trim(f[4]),
                        tags = trim(f[5]),
                        group = trim(f[6]),
                        favorite = trim(f[7]) == "1",
                        gitBranch = trim(f[8]),
                        gitDirty = tonumber(trim(f[9])) or 0,
                        remote = trim(f[10]),
                    })
                end
            end
        end
        cachedProjects = projects
        loadError = (result.exitCode ~= 0) and trim(result.stderr or result.stdout or "") or nil
        noctalia.log(string.format("pjx: list exit=%d projects=%d", result.exitCode, #projects))
        onDone(projects)
    end)
end

local function projectName(p)
    local name = p.name
    if name == "" then
        name = p.path:match("([^/]+)$") or p.path
    end
    return name
end

local function pathExists(path, onDone)
    noctalia.runAsync("test -d " .. shellQuote(path) .. " && echo yes || echo no", function(result)
        onDone(result.exitCode == 0 and trim(result.stdout) == "yes")
    end)
end

local function hasTag(p, tag)
    for t in (p.tags or ""):gmatch("[^,]+") do
        if trim(t):lower() == tag:lower() then
            return true
        end
    end
    return false
end

local function makeProjectRows(projects, filter)
    local home = noctalia.getenv("HOME") or ""
    local rows = {}
    -- Tag filter: query like "#Work" filters by tag.
    local tagFilter = nil
    local textFilter = filter
    if filter:sub(1, 1) == "#" then
        tagFilter = filter:sub(2)
        textFilter = ""
    end
    for _, p in ipairs(projects) do
        -- tag filter check
        if tagFilter and not hasTag(p, tagFilter) then
            continue
        end
        local name = projectName(p)
        local score = 0
        if textFilter ~= "" then
            score = noctalia.fuzzyScore(textFilter, name)
            if score == nil then
                score = noctalia.fuzzyScore(textFilter, p.path)
            end
            if score == nil then
                if
                    name:lower():find(textFilter:lower(), 1, true)
                    or p.path:lower():find(textFilter:lower(), 1, true)
                then
                    score = 1
                else
                    score = nil
                end
            end
        end
        if textFilter == "" or score ~= nil then
            local subtitle = p.path
            if home ~= "" and p.path:sub(1, #home) == home then
                subtitle = "~" .. p.path:sub(#home + 1)
            end
            if p.gitBranch ~= "" then
                subtitle = subtitle .. "  [" .. p.gitBranch
                if p.gitDirty > 0 then
                    subtitle = subtitle .. " +" .. tostring(p.gitDirty)
                end
                subtitle = subtitle .. "]"
            end
            local icon = "folder"
            if p.editor == "zed" then
                icon = "zed"
            elseif p.editor == "code" or p.editor == "codium" or p.editor == "cursor" then
                icon = "code"
            elseif p.editor == "nvim" or p.editor == "vim" then
                icon = "terminal"
            end
            if p.favorite then
                icon = "star"
            end
            table.insert(rows, {
                id = p.path,
                title = name,
                subtitle = subtitle,
                icon = icon,
                score = (textFilter == "") and nil or score,
            })
        end
    end
    return rows
end

local function utilRows(filter)
    local rows = {}
    local q = filter:lower()
    local function add(id, title, subtitle, glyph)
        if filter == "" or title:lower():find(q, 1, true) or subtitle:lower():find(q, 1, true) then
            table.insert(rows, { id = id, title = title, subtitle = subtitle, glyph = glyph, score = 1 })
        end
    end
    add("util:scan", "Scan directories for projects", "run pjx scan on configured roots", "folder-search")
    add("util:import-zed", "Import projects from Zed", "one-time migration", "download")
    add("util:import-all", "Import from all editors", "zed, vscode, cursor, ...", "download")
    add("util:editor-zed", "Set default editor: zed", "pjx config set default_editor zed", "code")
    add("util:editor-code", "Set default editor: vscode", "pjx config set default_editor code", "code")
    add("util:editor-nvim", "Set default editor: nvim", "pjx config set default_editor nvim", "code")
    add("util:refresh", "Refresh project list", "clear cache and reload", "refresh")
    add("util:profiles", "List VS Code profiles", "show known VS Code profiles", "user")
    -- Create profile from query text
    local qLower = (filter or ""):lower()
    if qLower:find("profile", 1, true) or qLower:find("prof", 1, true) then
        local name = nil
        for _, pat in ipairs({ "create%s+(.+)", "new%s+(.+)" }) do
            local m = filter:match(pat)
            if m and trim(m) ~= "" then
                name = trim(m)
                break
            end
        end
        if name then
            table.insert(rows, {
                id = "vscprofile:create:" .. name,
                title = "Create VS Code profile: " .. name,
                subtitle = "pjx profiles create " .. name,
                glyph = "plus",
                score = 100,
            })
        else
            table.insert(rows, {
                id = "vscprofile:createprompt",
                title = "Create VS Code profile",
                subtitle = "type: profile create <name>",
                glyph = "plus",
                score = 50,
            })
        end
    end
    return rows
end

local function showResults(query, projects, filter)
    local rows = makeProjectRows(projects, filter)

    -- If query looks like a path and exists on disk, offer to add it.
    local trimmedFilter = trim(filter)
    if
        trimmedFilter ~= ""
        and (trimmedFilter:find("/", 1, true) or trimmedFilter:sub(1, 1) == "~" or trimmedFilter:sub(1, 1) == ".")
    then
        local expanded = trimmedFilter
        if expanded:sub(1, 2) == "~/" then
            expanded = (noctalia.getenv("HOME") or "") .. expanded:sub(2)
        end
        local already = false
        for _, p in ipairs(projects) do
            if p.path == expanded then
                already = true
                break
            end
        end
        if not already then
            pathExists(expanded, function(exists)
                if exists then
                    local name = expanded:match("([^/]+)$") or expanded
                    local addRow = {
                        id = "add:" .. expanded,
                        title = "Add project: " .. name,
                        subtitle = expanded,
                        glyph = "plus",
                        score = 100,
                    }
                    table.insert(rows, 1, addRow)
                    launcher.setResults(query, rows)
                else
                    launcher.setResults(query, rows)
                end
            end)
            return
        end
    end

    -- Utility entries (searchable by title/subtitle).
    for _, r in ipairs(utilRows(filter)) do
        table.insert(rows, r)
    end

    -- pjx failed: surface the error instead of an empty list.
    if loadError ~= nil and loadError ~= "" then
        launcher.setResults(query, {
            {
                id = "",
                title = "pjx error",
                subtitle = loadError:sub(1, 120),
                glyph = "alert-triangle",
            },
        })
        return
    end

    if #rows == 0 then
        launcher.setResults(query, {
            {
                id = "",
                title = noctalia.tr("no-projects-found") or "No projects found",
                subtitle = (filter ~= "") and ("no match for: " .. filter) or "run `pjx scan` to discover projects",
                glyph = "folder-x",
            },
        })
    else
        launcher.setResults(query, rows)
    end
end

function onQuery(query)
    local filter = trim(query)
    noctalia.log("pjx: onQuery '" .. filter .. "'")

    if filter == "" then
        cachedProjects = nil
    end

    if cachedProjects then
        showResults(query, cachedProjects, filter)
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
        showResults(query, projects, filter)
    end)
end

function onActivate(id)
    noctalia.log("pjx: onActivate '" .. id .. "'")
    if id == "" then
        return
    end
    if id:match("^add:") then
        local path = id:sub(5)
        noctalia.runAsync("pjx add " .. shellQuote(path) .. " 2>&1", function(result)
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
            noctalia.runAsync("pjx profiles create " .. shellQuote(name) .. " 2>&1", function(result)
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
            noctalia.runAsync("pjx scan 2>&1", function(result)
                noctalia.notify("projx", "Scan complete")
                cachedProjects = nil
            end)
        elseif action == "import-zed" then
            noctalia.runAsync("pjx import zed 2>&1", function(result)
                noctalia.notify("projx", "Imported from Zed")
                cachedProjects = nil
            end)
        elseif action == "import-all" then
            noctalia.runAsync("pjx import --all 2>&1", function(result)
                noctalia.notify("projx", "Import complete")
                cachedProjects = nil
            end)
        elseif action == "refresh" then
            cachedProjects = nil
            noctalia.notify("projx", "Refreshed")
        elseif action == "profiles" then
            noctalia.runAsync("pjx profiles 2>&1", function(result)
                local out = trim(result.stdout or "")
                if out == "" then
                    out = trim(result.stderr or "")
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
            noctalia.runAsync("pjx config set default_editor " .. ed .. " 2>&1", function(result)
                noctalia.notify("projx", "Default editor set to " .. ed)
            end)
        end
        return
    end
    -- normal project open
    local newWindow = noctalia.getConfig("new_window") == true
    local args = "pjx open " .. shellQuote(id)
    if newWindow then
        args = args .. " --new-window"
    end
    noctalia.runAsync(args .. " 2>&1", function(result)
        noctalia.log(string.format("pjx: open exit=%d", result.exitCode))
        if result.exitCode ~= 0 then
            noctalia.notifyError(
                "Failed to open project",
                trim(result.stderr or result.stdout or "unknown error"):sub(1, 160)
            )
        end
    end)
end
