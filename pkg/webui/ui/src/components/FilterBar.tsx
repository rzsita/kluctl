import React, { useState } from "react";
import { Box, Chip } from "@mui/material";
import { AutoFixHigh, ErrorOutline, Grade } from "@mui/icons-material";
import { OverridableComponent } from "@mui/material/OverridableComponent";
import Tooltip from "@mui/material/Tooltip";
import { useTheme } from "@mui/material/styles";

export interface ActiveFilters {
    onlyImportant: boolean
    onlyChanged: boolean
    onlyWithErrorsOrWarnings: boolean
    filterStr: string
}

export function DoFilterSwitches(hasChanges: boolean, hasErrors: boolean, hasWarnings: boolean, activeFilters?: ActiveFilters) {
    const hasErrorsOrWarnings = hasErrors || hasWarnings
    if (activeFilters?.onlyImportant && !hasChanges && !hasErrorsOrWarnings) {
        return false
    }
    if (activeFilters?.onlyChanged && !hasChanges) {
        return false
    }
    if (activeFilters?.onlyWithErrorsOrWarnings && !hasErrorsOrWarnings) {
        return false
    }
    return true
}

export function DoFilterText(text: (string|undefined)[] | null, activeFilters?: ActiveFilters) {
    if (activeFilters?.filterStr && text && text.length) {
        const l = activeFilters.filterStr.toLowerCase()
        const f = text.find(t => t && t.toLowerCase().includes(l))
        if (!f) {
            return false
        }
    }
    return true
}

const FilterButton = (props: { Icon: OverridableComponent<any>, tooltip: string, color: any, active: boolean, handler: (active: boolean) => void }) => {
    const handleClick = () => {
        props.handler(!props.active)
    }

    const Icon = props.Icon
    const chipColor = props.active ? props.color : "default";
    return <Tooltip title={props.tooltip}>
        <Chip
            variant="filled"
            color={chipColor}
            label={
                <Icon
                    color={props.active ? undefined : props.color}
                    htmlColor={props.active ? "white" : undefined}
                />
            }
            onClick={handleClick}
            sx={{
                "& .MuiChip-label": {
                    display: "flex",
                    justifyContent: "center",
                    alignItems: "center"
                }
            }}
        />
    </Tooltip>
}

export const FilterBar = (props: { onFilterChange: (f: ActiveFilters) => void }) => {
    const theme = useTheme();

    const [activeFilters, setActiveFilters] = useState<ActiveFilters>({
        onlyImportant: false,
        onlyChanged: false,
        onlyWithErrorsOrWarnings: false,
        filterStr: "",
    })

    const doSetActiveFilters = (newActiveFilters: ActiveFilters) => {
        setActiveFilters(newActiveFilters)
        props.onFilterChange(newActiveFilters)
    }

    return (
        <Box display={"flex"} flexDirection={"column"} alignItems={"center"}>
            <Box
                display="flex"
                alignItems="center"
                gap="5px"
            >
                <FilterButton Icon={Grade} tooltip={"Only important (changed or with errors/warnings)"}
                              color={"secondary"}
                              active={activeFilters.onlyImportant}
                              handler={(active: boolean) => {
                                  doSetActiveFilters({ ...activeFilters, onlyImportant: active });
                              }}/>
                <FilterButton Icon={AutoFixHigh} tooltip={"Only with changed"} color={"secondary"}
                              active={activeFilters.onlyChanged}
                              handler={(active: boolean) => {
                                  doSetActiveFilters({ ...activeFilters, onlyChanged: active });
                              }}/>
                <FilterButton Icon={ErrorOutline} tooltip={"Only with errors or warnings"} color={"error"}
                              active={activeFilters.onlyWithErrorsOrWarnings}
                              handler={(active: boolean) => {
                                  doSetActiveFilters({ ...activeFilters, onlyWithErrorsOrWarnings: active });
                              }}/>
                <Box
                    height='40px'
                    maxWidth='314px'
                    flexGrow={1}
                    borderRadius='10px'
                    display='flex'
                    justifyContent='space-between'
                    alignItems='center'
                    padding='0 9px 0 15px'
                    sx={{ background: theme.palette.background.default }}
                >
                    <input
                        type='text'
                        style={{
                            background: 'none',
                            border: 'none',
                            outline: 'none',
                            height: '20px',
                            lineHeight: '20px',
                            fontSize: '18px'
                        }}
                        placeholder='Filter'
                        onChange={e => {
                            doSetActiveFilters({ ...activeFilters, filterStr: e.target.value })
                        }}
                    />
                </Box>
            </Box>
        </Box>
    )
}