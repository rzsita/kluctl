import React, { useContext, useMemo } from "react";
import { ProjectSummary, TargetSummary } from "../../project-summaries";
import { CommandResultSummary, ShortName } from "../../models";
import { Box, IconButton, SxProps, Theme, Tooltip } from "@mui/material";
import { useNavigate } from "react-router";
import { DeployIcon, DiffIcon, PruneIcon, TreeViewIcon } from "../../icons/Icons";
import { LiveHelp, RocketLaunch } from "@mui/icons-material";
import * as yaml from "js-yaml";
import { CodeViewer } from "../CodeViewer";
import { CardTemplate } from "./Card";
import { Since } from "../Since";
import { CommandResultStatusLine } from "../result-view/CommandResultStatusLine";
import { CommandResultSummaryBody } from "./CommandResultSummaryView";
import { ApiContext, UserContext } from "../App";
import { Api } from "../../api";
import { NodeBuilder } from "../result-view/nodes/NodeBuilder";

export async function doGetRootNode(api: Api, rs: CommandResultSummary, shortNames: ShortName[]) {
    const r = await api.getCommandResult(rs.id);
    const builder = new NodeBuilder({
        shortNames,
        commandResult: r,
    });
    const [node] = builder.buildRoot();
    return node;
}

const ApprovalIcon = (props: {ts: TargetSummary, rs: CommandResultSummary}) => {
    const api = useContext(ApiContext)
    const user = useContext(UserContext)
    const handleApprove = (approve: boolean) => {
        if (!props.ts.kdInfo || !props.ts.kd) {
            return
        }
        if (approve) {
            if (!props.rs.renderedObjectsHash) {
                return
            }
            api.setManualObjectsHash(props.ts.kdInfo.clusterId, props.ts.kdInfo.name, props.ts.kdInfo.namespace, props.rs.renderedObjectsHash)
        } else {
            api.setManualObjectsHash(props.ts.kdInfo.clusterId, props.ts.kdInfo.name, props.ts.kdInfo.namespace, "")
        }
    }

    if (!user?.isAdmin || props.ts.kd?.deployment.spec.dryRun || !props.ts.kd?.deployment.spec.manual) {
        return <></>
    }
    if (props.rs.id !== props.ts.commandResults[0].id) {
        return <></>
    }

    if (!props.rs.commandInfo.dryRun || !props.rs.renderedObjectsHash) {
        return <></>
    }

    const isApproved = props.ts.kd.deployment.spec.manualObjectsHash === props.rs.renderedObjectsHash

    let icon: React.ReactElement
    let tooltip: string
    if (!isApproved) {
        tooltip = "Click here to trigger this manual deployment."
        icon = <RocketLaunch color={"info"}/>
    } else {
        tooltip = "Click here to cancel this deployment. This will only have an effect if the deployment has not started reconciliation yet!"
        icon = <RocketLaunch color={"success"}/>
    }
    return <Box display='flex' gap='6px' alignItems='center' height='39px'>
        <IconButton
            onClick={e => {
                e.stopPropagation();
                handleApprove(!isApproved)
            }}
            sx={{
                padding: 0,
                width: 26,
                height: 26
            }}
        >
            <Tooltip title={tooltip}>
                <Box display='flex'>{icon}</Box>
            </Tooltip>
        </IconButton>
    </Box>
}

export const CommandResultCard = React.memo(React.forwardRef((
    props: {
        current: boolean,
        ps: ProjectSummary,
        ts: TargetSummary,
        rs: CommandResultSummary,
        onSelectCommandResult?: (rs: CommandResultSummary) => void,
        sx?: SxProps<Theme>,
        showSummary: boolean,
        expanded?: boolean,
        loadData?: boolean,
        onClose?: () => void
    },
    ref: React.ForwardedRef<HTMLDivElement>
) => {
    const navigate = useNavigate();

    let icon: React.ReactElement
    let cardGlow = false
    let header = props.rs.commandInfo?.command
    switch (props.rs.commandInfo?.command) {
        default:
            icon = <DiffIcon/>
            break
        case "delete":
            icon = <PruneIcon/>
            break
        case "deploy":
            if (props.rs.commandInfo.dryRun) {
                if (props.ts.kd?.deployment.spec.manual && !props.ts.kd?.deployment.spec.dryRun) {
                    icon = <LiveHelp sx={{ width: "100%", height: "100%" }}/>
                    cardGlow = true
                    header = "manual deploy"
                } else {
                    icon = <DeployIcon/>
                    header = "dry-run deploy"
                }
            } else {
                icon = <DeployIcon/>
            }
            break
        case "diff":
            icon = <DiffIcon/>
            break
        case "poke-images":
            icon = <DeployIcon/>
            break
        case "prune":
            icon = <PruneIcon/>
            break
    }

    const iconTooltip = useMemo(() => {
        const cmdInfoYaml = yaml.dump(props.rs.commandInfo);
        return <CodeViewer code={cmdInfoYaml} language={"yaml"} />
    }, [props.rs.commandInfo]);

    let body: React.ReactElement | undefined
    if (props.expanded) {
        if (props.showSummary) {
            body = <CommandResultSummaryBody rs={props.rs} loadData={props.loadData} />
        }
    }

    const footer = <>
        <Box display='flex' gap='6px' alignItems='center' flex={"1 1 auto"}>
            <CommandResultStatusLine rs={props.rs} />
        </Box>
        <ApprovalIcon ts={props.ts} rs={props.rs}/>
        <Box display='flex' gap='6px' alignItems='center' height='39px'>
            <IconButton
                onClick={e => {
                    e.stopPropagation();
                    navigate(`/results/${props.rs.id}`);
                }}
                sx={{
                    padding: 0,
                    width: 26,
                    height: 26
                }}
            >
                <Tooltip title={"Open Result Tree"}>
                    <Box display='flex'><TreeViewIcon /></Box>
                </Tooltip>
            </IconButton>
        </Box>
    </>

    return <CardTemplate
        ref={ref}
        showCloseButton={props.expanded}
        onClose={props.onClose}
        paperProps={{
            sx: {
                padding: '20px 16px 5px 16px',
                ...props.sx,
            },
            glow: cardGlow,
            onClick: () => props.onSelectCommandResult?.(props.rs)
        }}
        icon={icon}
        iconTooltip={iconTooltip}
        header={header}
        subheader={<Since startTime={new Date(props.rs.commandInfo.startTime)}/>}
        subheaderTooltip={props.rs.commandInfo.startTime}
        body={body}
        footer={footer}
    />;
}));