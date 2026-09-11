<?php
require_once __DIR__ . "/demandreviewresponse.php";

class taskStatusEntry extends entry
{
    public function post($taskID)
    {
        $taskID = (int)$taskID;
        if($taskID <= 0) return $this->sendError(400, 'Invalid task id');

        $data = json_decode(file_get_contents('php://input'));
        if(!is_object($data)) $data = new stdclass();
        $target = isset($data->status) ? trim((string)$data->status) : '';
        if(!in_array($target, array('wait', 'doing', 'done'), true)) return $this->sendError(400, 'Invalid task status');

        $task = $this->dao->select('id,status,assignedTo')->from(TABLE_TASK)
            ->where('id')->eq($taskID)
            ->andWhere('deleted')->eq('0')
            ->fetch();
        if(!$task) return $this->sendError(404, 'Task not found');

        $now = helper::now();
        $account = isset($this->app->user->account) ? $this->app->user->account : '';
        $updates = new stdclass();
        $updates->status = $target;
        $updates->lastEditedBy = $account;
        $updates->lastEditedDate = $now;

        $action = 'edited';
        $comment = '';
        if($target === 'done')
        {
            $finishedBy = isset($data->finishedBy) ? trim((string)$data->finishedBy) : '';
            if($finishedBy === '') $finishedBy = trim((string)$task->assignedTo);
            if($finishedBy === '') $finishedBy = $account;
            $finishedDate = isset($data->finishedDate) ? trim((string)$data->finishedDate) : '';
            if($finishedDate === '') $finishedDate = $now;

            $updates->left = 0;
            $updates->finishedBy = $finishedBy;
            $updates->finishedDate = $finishedDate;
            $action = 'finished';
            $comment = 'Workbench task board drag to done.';
        }
        else
        {
            $updates->finishedBy = '';
            $updates->finishedDate = '0000-00-00 00:00:00';
            $action = $target === 'doing' ? 'started' : 'activated';
            $comment = 'Workbench task board drag to ' . $target . '.';
        }

        $this->dao->update(TABLE_TASK)->data($updates)
            ->where('id')->eq($taskID)
            ->andWhere('deleted')->eq('0')
            ->andWhere('status')->eq($task->status)
            ->exec();
        if(dao::isError()) return $this->sendError(400, dao::getError());

        $this->loadModel('action')->create('task', $taskID, $action, $comment);
        if(dao::isError()) return $this->sendError(400, dao::getError());

        return $this->send(200, array('success' => true, 'message' => 'Success'));
    }
}
