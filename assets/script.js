const confirm_delete = (id) => {
    if (window.confirm(`Task ${id} を削除します．よろしいですか？`)) {
        location.href = `/task/delete/${id}`;
    }
}

const confirm_update = (id) => {
    return window.confirm(`Task ${id} を更新します．よろしいですか？`);
}

