-- Controlling Aseprite Slice pivots is very difficult.
-- Calculate the pivot positions from another layer bounds and write them to the Tag's UserData.
-- Examine the pivots layer in the bird.ase file. You can adjust the pivot point using the Move Tool.

local sprite = app.activeSprite
local pivotLayer = nil

for _, layer in ipairs(sprite.layers) do
    if layer.name == "pivots" then
        pivotLayer = layer
        break
    end
end

for _, tag in ipairs(sprite.tags) do
    local pivotLayer = pivotLayer:cel(tag.fromFrame)
    local x = pivotLayer.bounds.x + 8
    local y = pivotLayer.bounds.y + 8
    tag.data = x .. "," .. y
end
