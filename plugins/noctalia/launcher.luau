--!nonstrict

-- Project Manager launcher provider for Noctalia, backed by the pjx CLI.
-- Supports: opening projects, adding new projects from a path, and quick
-- configuration actions (scan, import, set default editor).

local cachedProjects = nil
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
	local cmd = "pjx list --format=delimited 2>/dev/null"
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

local function makeProjectRows(projects, filter)
	local home = noctalia.getenv("HOME") or ""
	local rows = {}
	for _, p in ipairs(projects) do
		local name = projectName(p)
		local score = 0
		if filter ~= "" then
			score = noctalia.fuzzyScore(filter, name)
			if score == nil then
				score = noctalia.fuzzyScore(filter, p.path)
			end
			if score == nil then
				if name:lower():find(filter:lower(), 1, true) or p.path:lower():find(filter:lower(), 1, true) then
					score = 1
				else
					score = nil
				end
			end
		end
		if filter == "" or score ~= nil then
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
			if p.favorite then
				icon = "star"
			end
			table.insert(rows, {
				id = p.path,
				title = name,
				subtitle = subtitle,
				icon = icon,
				score = (filter == "") and nil or score,
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
		-- check if already a project
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

	-- Utility entries: show when filter empty or matches config/scan/import keywords.
	local lower = filter:lower()
	local showUtil = (filter == "")
		or lower:find("scan", 1, true)
		or lower:find("import", 1, true)
		or lower:find("config", 1, true)
		or lower:find("settings", 1, true)
		or lower:find("editor", 1, true)
		or lower:find("add", 1, true)
		or lower:find("help", 1, true)
	if showUtil then
		for _, r in ipairs(utilRows(filter)) do
			table.insert(rows, r)
		end
	end

	if #rows == 0 then
		launcher.setResults(query, {
			{
				id = "",
				title = "No projects found",
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
			title = "Loading projects...",
			subtitle = "querying pjx",
			glyph = "loader",
		},
	})

	loadProjects(function(projects)
		showResults(query, projects, filter)
	end)
end

function onActivate(id)
	if id == "" then
		return
	end
	if id:sub(1, 4) == "add:" then
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
	if id:sub(1, 6) == "util:" then
		local action = id:sub(7)
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
	local cmd
	if newWindow then
		cmd = string.format("nohup pjx open %s --new-window >/dev/null 2>&1 &", shellQuote(id))
	else
		cmd = string.format("nohup pjx open %s >/dev/null 2>&1 &", shellQuote(id))
	end
	noctalia.runAsync(cmd)
end
