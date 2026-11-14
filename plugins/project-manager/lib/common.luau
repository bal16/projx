--!nonstrict
-- Shared helpers for the projx Noctalia plugin.

local M = {}
local FIELD_SEP = "\31"

function M.trim(s)
    return (s or ""):match("^%s*(.-)%s*$")
end

function M.shellQuote(s)
    return "'" .. tostring(s):gsub("'", "'\"'\"'") .. "'"
end

function M.splitFields(line)
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

function M.parseProjects(stdout)
    local projects = {}
    for line in (stdout or ""):gmatch("[^\n]+") do
        local f = M.splitFields(line)
        local id = M.trim(f[1])
        if id ~= "" then
            table.insert(projects, {
                id = id,
                name = M.trim(f[2]),
                path = M.trim(f[3]),
                editor = M.trim(f[4]),
                tags = M.trim(f[5]),
                group = M.trim(f[6]),
                favorite = M.trim(f[7]) == "1",
                gitBranch = M.trim(f[8]),
                gitDirty = tonumber(M.trim(f[9])) or 0,
                remote = M.trim(f[10]),
            })
        end
    end
    return projects
end

function M.projectName(p)
    local name = p.name
    if name == "" then
        name = p.path:match("([^/]+)$") or p.path
    end
    return name
end

function M.hasTag(p, tag)
    for t in (p.tags or ""):gmatch("[^,]+") do
        if M.trim(t):lower() == tag:lower() then
            return true
        end
    end
    return false
end

function M.shortPath(path)
    local home = noctalia.getenv("HOME") or ""
    if home ~= "" and path:sub(1, #home) == home then
        return "~" .. path:sub(#home + 1)
    end
    return path
end

function M.pathExists(path, onDone)
    noctalia.runAsync("test -d " .. M.shellQuote(path) .. " && echo yes || echo no", function(result)
        onDone(result.exitCode == 0 and M.trim(result.stdout) == "yes")
    end)
end

return M
