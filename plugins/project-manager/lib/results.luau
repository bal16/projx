--!nonstrict
-- Row/result builders for the projx launcher provider.

local common = require("./common.luau")
local M = {}

function M.projectRows(projects, filter)
    local rows = {}
    -- Tag filter: query like "#Work" filters by tag.
    local tagFilter = nil
    local textFilter = filter
    if filter:sub(1, 1) == "#" then
        tagFilter = filter:sub(2)
        textFilter = ""
    end
    for _, p in ipairs(projects) do
        if not (tagFilter and not common.hasTag(p, tagFilter)) then
            local name = common.projectName(p)
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
                local subtitle = common.shortPath(p.path)
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
    end
    return rows
end

function M.utilRows(filter)
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

    local qLower = (filter or ""):lower()
    if qLower:find("profile", 1, true) or qLower:find("prof", 1, true) then
        local name = nil
        for _, pat in ipairs({ "create%s+(.+)", "new%s+(.+)" }) do
            local m = filter:match(pat)
            if m and common.trim(m) ~= "" then
                name = common.trim(m)
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

function M.showResults(query, projects, filter, loadError)
    local rows = M.projectRows(projects, filter)

    -- If query looks like a path and exists on disk, offer to add it.
    local trimmedFilter = common.trim(filter)
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
            common.pathExists(expanded, function(exists)
                if exists then
                    local name = expanded:match("([^/]+)$") or expanded
                    table.insert(rows, 1, {
                        id = "add:" .. expanded,
                        title = "Add project: " .. name,
                        subtitle = expanded,
                        glyph = "plus",
                        score = 100,
                    })
                end
                launcher.setResults(query, rows)
            end)
            return
        end
    end

    -- Utility entries (searchable by title/subtitle).
    for _, r in ipairs(M.utilRows(filter)) do
        table.insert(rows, r)
    end

    -- pjx failed: surface the error instead of an empty list.
    if loadError ~= nil and loadError ~= "" then
        launcher.setResults(query, {
            { id = "", title = "pjx error", subtitle = loadError:sub(1, 120), glyph = "alert-triangle" },
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

return M
