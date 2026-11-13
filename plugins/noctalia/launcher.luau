--!nonstrict

-- Project Manager launcher provider for Noctalia, backed by the pjx CLI.

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
				-- fields: id, name, path, editor, tags, group, favorite, git_branch, git_dirty, remote
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

local function makeRows(projects, filter)
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
				-- fallback: substring
				if not (name:lower():find(filter:lower(), 1, true) or p.path:lower():find(filter:lower(), 1, true)) then
					score = nil
				else
					score = 1
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

local function showResults(query, projects, filter)
	local rows = makeRows(projects, filter)
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
	local newWindow = noctalia.getConfig("new_window") == true
	local cmd
	if newWindow then
		cmd = string.format("nohup pjx open %s --new-window >/dev/null 2>&1 &", shellQuote(id))
	else
		cmd = string.format("nohup pjx open %s >/dev/null 2>&1 &", shellQuote(id))
	end
	noctalia.runAsync(cmd)
end
