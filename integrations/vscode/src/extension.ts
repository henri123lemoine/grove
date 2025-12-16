import * as vscode from 'vscode';
import * as cp from 'child_process';
import * as path from 'path';

interface Worktree {
    path: string;
    branch: string;
    isCurrent: boolean;
    isDirty: boolean;
}

class WorktreeItem extends vscode.TreeItem {
    constructor(
        public readonly worktree: Worktree,
        public readonly collapsibleState: vscode.TreeItemCollapsibleState
    ) {
        super(worktree.branch || '(detached)', collapsibleState);

        this.description = worktree.path;
        this.tooltip = `${worktree.branch}\n${worktree.path}`;

        if (worktree.isCurrent) {
            this.contextValue = 'current';
            this.iconPath = new vscode.ThemeIcon('check');
        } else {
            this.contextValue = 'worktree';
            this.iconPath = new vscode.ThemeIcon('git-branch');
        }

        if (worktree.isDirty) {
            this.description += ' (modified)';
        }

        this.command = {
            command: 'grove.openWorktree',
            title: 'Open Worktree',
            arguments: [worktree]
        };
    }
}

class WorktreeProvider implements vscode.TreeDataProvider<WorktreeItem> {
    private _onDidChangeTreeData = new vscode.EventEmitter<WorktreeItem | undefined | null | void>();
    readonly onDidChangeTreeData = this._onDidChangeTreeData.event;

    refresh(): void {
        this._onDidChangeTreeData.fire();
    }

    getTreeItem(element: WorktreeItem): vscode.TreeItem {
        return element;
    }

    async getChildren(): Promise<WorktreeItem[]> {
        const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
        if (!workspaceFolder) {
            return [];
        }

        try {
            const worktrees = await this.getWorktrees(workspaceFolder.uri.fsPath);
            return worktrees.map(wt => new WorktreeItem(wt, vscode.TreeItemCollapsibleState.None));
        } catch (error) {
            console.error('Failed to get worktrees:', error);
            return [];
        }
    }

    private async getWorktrees(cwd: string): Promise<Worktree[]> {
        return new Promise((resolve, reject) => {
            cp.exec('git worktree list --porcelain', { cwd }, (err, stdout) => {
                if (err) {
                    reject(err);
                    return;
                }

                const worktrees: Worktree[] = [];
                const lines = stdout.split('\n');
                let current: Partial<Worktree> = {};

                for (const line of lines) {
                    if (line.startsWith('worktree ')) {
                        if (current.path) {
                            worktrees.push(current as Worktree);
                        }
                        current = {
                            path: line.substring(9),
                            isDirty: false,
                            isCurrent: false
                        };
                    } else if (line.startsWith('branch ')) {
                        current.branch = line.substring(7).replace('refs/heads/', '');
                    } else if (line === 'detached') {
                        current.branch = '(detached)';
                    }
                }

                if (current.path) {
                    worktrees.push(current as Worktree);
                }

                // Mark current worktree
                const currentPath = cwd;
                for (const wt of worktrees) {
                    if (wt.path === currentPath || currentPath.startsWith(wt.path + path.sep)) {
                        wt.isCurrent = true;
                        break;
                    }
                }

                resolve(worktrees);
            });
        });
    }
}

export function activate(context: vscode.ExtensionContext) {
    const worktreeProvider = new WorktreeProvider();

    vscode.window.registerTreeDataProvider('groveWorktrees', worktreeProvider);

    // Refresh worktrees view
    context.subscriptions.push(
        vscode.commands.registerCommand('grove.refresh', () => {
            worktreeProvider.refresh();
        })
    );

    // Open worktree
    context.subscriptions.push(
        vscode.commands.registerCommand('grove.openWorktree', async (worktree?: Worktree) => {
            if (!worktree) {
                const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
                if (!workspaceFolder) {
                    vscode.window.showErrorMessage('No workspace folder open');
                    return;
                }

                // Show quick pick
                const items = await worktreeProvider.getChildren();
                const selected = await vscode.window.showQuickPick(
                    items.map(item => ({
                        label: item.worktree.branch,
                        description: item.worktree.path,
                        worktree: item.worktree
                    })),
                    { placeHolder: 'Select a worktree to open' }
                );

                if (!selected) return;
                worktree = selected.worktree;
            }

            // Open in new window
            const uri = vscode.Uri.file(worktree.path);
            await vscode.commands.executeCommand('vscode.openFolder', uri, { forceNewWindow: true });
        })
    );

    // Create worktree
    context.subscriptions.push(
        vscode.commands.registerCommand('grove.createWorktree', async () => {
            const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
            if (!workspaceFolder) {
                vscode.window.showErrorMessage('No workspace folder open');
                return;
            }

            const branchName = await vscode.window.showInputBox({
                prompt: 'Enter branch name for new worktree',
                placeHolder: 'feature/my-feature'
            });

            if (!branchName) return;

            // Check if branch exists
            const branchExists = await new Promise<boolean>((resolve) => {
                cp.exec(`git rev-parse --verify refs/heads/${branchName}`,
                    { cwd: workspaceFolder.uri.fsPath },
                    (err) => resolve(!err)
                );
            });

            const worktreePath = path.join(
                workspaceFolder.uri.fsPath,
                '.worktrees',
                branchName.replace(/\//g, '-')
            );

            const args = branchExists
                ? `git worktree add "${worktreePath}" "${branchName}"`
                : `git worktree add -b "${branchName}" "${worktreePath}"`;

            cp.exec(args, { cwd: workspaceFolder.uri.fsPath }, (err, stdout, stderr) => {
                if (err) {
                    vscode.window.showErrorMessage(`Failed to create worktree: ${stderr || err.message}`);
                    return;
                }

                vscode.window.showInformationMessage(`Created worktree: ${branchName}`);
                worktreeProvider.refresh();

                // Offer to open the new worktree
                vscode.window.showInformationMessage(
                    `Open worktree ${branchName}?`,
                    'Open',
                    'Cancel'
                ).then(selection => {
                    if (selection === 'Open') {
                        vscode.commands.executeCommand('vscode.openFolder',
                            vscode.Uri.file(worktreePath),
                            { forceNewWindow: true }
                        );
                    }
                });
            });
        })
    );

    // Delete worktree
    context.subscriptions.push(
        vscode.commands.registerCommand('grove.deleteWorktree', async (item?: WorktreeItem) => {
            const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
            if (!workspaceFolder) {
                vscode.window.showErrorMessage('No workspace folder open');
                return;
            }

            let worktree: Worktree | undefined;

            if (item) {
                worktree = item.worktree;
            } else {
                // Show quick pick
                const items = await worktreeProvider.getChildren();
                const nonCurrentItems = items.filter(i => !i.worktree.isCurrent);

                if (nonCurrentItems.length === 0) {
                    vscode.window.showInformationMessage('No worktrees to delete (cannot delete current worktree)');
                    return;
                }

                const selected = await vscode.window.showQuickPick(
                    nonCurrentItems.map(item => ({
                        label: item.worktree.branch,
                        description: item.worktree.path,
                        worktree: item.worktree
                    })),
                    { placeHolder: 'Select a worktree to delete' }
                );

                if (!selected) return;
                worktree = selected.worktree;
            }

            if (worktree.isCurrent) {
                vscode.window.showErrorMessage('Cannot delete current worktree');
                return;
            }

            const confirm = await vscode.window.showWarningMessage(
                `Delete worktree "${worktree.branch}"?`,
                { modal: true },
                'Delete'
            );

            if (confirm !== 'Delete') return;

            cp.exec(`git worktree remove "${worktree.path}"`,
                { cwd: workspaceFolder.uri.fsPath },
                (err, stdout, stderr) => {
                    if (err) {
                        // Try force remove
                        vscode.window.showWarningMessage(
                            `Worktree has uncommitted changes. Force delete?`,
                            'Force Delete',
                            'Cancel'
                        ).then(selection => {
                            if (selection === 'Force Delete') {
                                cp.exec(`git worktree remove --force "${worktree!.path}"`,
                                    { cwd: workspaceFolder.uri.fsPath },
                                    (err2) => {
                                        if (err2) {
                                            vscode.window.showErrorMessage(`Failed to delete worktree: ${err2.message}`);
                                        } else {
                                            vscode.window.showInformationMessage(`Deleted worktree: ${worktree!.branch}`);
                                            worktreeProvider.refresh();
                                        }
                                    }
                                );
                            }
                        });
                        return;
                    }

                    vscode.window.showInformationMessage(`Deleted worktree: ${worktree!.branch}`);
                    worktreeProvider.refresh();
                }
            );
        })
    );

    // List worktrees (quick pick)
    context.subscriptions.push(
        vscode.commands.registerCommand('grove.listWorktrees', async () => {
            await vscode.commands.executeCommand('grove.openWorktree');
        })
    );

    // Open grove in terminal
    context.subscriptions.push(
        vscode.commands.registerCommand('grove.openTerminal', () => {
            const terminal = vscode.window.createTerminal('Grove');
            terminal.sendText('grove');
            terminal.show();
        })
    );

    // Refresh on window focus
    context.subscriptions.push(
        vscode.window.onDidChangeWindowState(e => {
            if (e.focused) {
                worktreeProvider.refresh();
            }
        })
    );
}

export function deactivate() {}
