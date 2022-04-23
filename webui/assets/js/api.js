let Api = {
    Action: {
        Pool: {
            CreatePool: function (form) {
                form.json = true
                CallAPI({
                    title: "创建矿池",
                    icon: "info",
                    confirmButtonColor: '#3085d6',
                    cancelButtonColor: '#d33',
                }, {
                    Url: "/api/v1/pool/create",
                    Method: "POST"
                }, form, "/pool")
            },
            EditPool: function (form) {
                form.json = true
                CallAPI({
                    title: "修改矿池",
                    text: "注意：此操作会重启矿池并断开所有矿机！",
                    icon: "warning",
                    confirmButtonColor: '#d33',
                    cancelButtonColor: '#3085d6',
                }, {
                    Url: "/api/v1/pool/edit",
                    Method: "POST"
                }, form, "/pool")
            },
            Delete: function (name) {
                CallAPI({
                    title: "永久删除矿池",
                    icon: "warning",
                    text: "注意：此操作不可逆，删除后将不可恢复！",
                    confirmButtonColor: '#d33',
                    cancelButtonColor: '#3085d6',
                }, {
                    Url: "/api/v1/pool/delete/" + name,
                    Method: "GET"
                }, {}, true)
            },
            Power: function (name, action) {
                if (action === "start") {
                    CallAPI({
                        title: "启动矿池",
                        icon: "info",
                        confirmButtonColor: '#00ffa6',
                        cancelButtonColor: '#3085d6',
                    }, {
                        Url: "/api/v1/pool/power/start/" + name,
                        Method: "GET"
                    }, {}, true)
                }

                if (action === "stop") {
                    CallAPI({
                        title: "关闭矿池",
                        icon: "warning",
                        text: "注意：此操作会导致连接此矿池下的所有矿机掉线！",
                        confirmButtonColor: '#d33',
                        cancelButtonColor: '#3085d6',
                    }, {
                        Url: "/api/v1/pool/power/stop/" + name,
                        Method: "GET"
                    }, {}, true)
                }
            },
        },
        Cfg: {
            AuthEdit: function (form) {
                form.json = true
                CallAPI({
                    title: "修改管理员认证信息",
                    icon: "info",
                    confirmButtonColor: '#00ffa6',
                    cancelButtonColor: '#3085d6',
                }, {
                    Url: "/api/v1/cfg/auth",
                    Method: "POST"
                }, form, true)
            }
        }
    }
}

function CallAPI(confirmAlert, apiInfo, data, redirect) {
    if (confirmAlert.confirmButtonColor === undefined) {
        confirmAlert.confirmButtonColor = "#d33"
    }
    if (confirmAlert.cancelButtonColor === undefined) {
        confirmAlert.confirmButtonColor = "#3085d6"
    }
    if (confirmAlert.text === undefined) {
        confirmAlert.text = ""
    }

    let resultHandler = function (result) {
        if (result.isConfirmed) {
            if (result.value.Result) {
                Swal.fire({
                    icon: 'success',
                    title: '成功',
                    text: result.value.Msg,
                }).then(function () {
                    if (redirect === true) {
                        document.location.reload()
                    } else if (redirect !== "") {
                        document.location.href = redirect
                    }
                })
            } else {
                Swal.fire({
                    icon: 'error',
                    title: '失败',
                    text: result.value.Msg,
                })
            }
        }
    }

    let ajaxPostForm = function (form) {
        return $.ajax({
            url: apiInfo.Url,
            method: apiInfo.Method,
            data: form,

            success: function (data) {
                return data
            },
            error: function (xhr, status) {
                return {
                    Result: false,
                    Msg: status,
                }
            }
        })
    }

    let ajaxPostJson = function (form) {
        return  $.ajax({
            url: apiInfo.Url,
            method: apiInfo.Method,
            data : JSON.stringify(form),
            contentType : 'application/json',
            processData: false,

            success: function (data) {
                return data
            },
            error: function (xhr, status) {
                return {
                    Result: false,
                    Msg: status,
                }
            }
        })
    }

    Swal.fire({
        title: confirmAlert.title,
        text: confirmAlert.text,
        confirmButtonText: '确认',
        cancelButtonText: '取消',
        showLoaderOnConfirm: true,
        showCancelButton: true,
        icon: confirmAlert.icon,
        confirmButtonColor: confirmAlert.confirmButtonColor,
        cancelButtonColor: confirmAlert.cancelButtonColor,
        preConfirm: () => {
            if (data.json) {
                return ajaxPostJson(data)
            }
            return ajaxPostForm(data)
        },
        allowOutsideClick: () => !Swal.isLoading(),
        backdrop: true,
    }).then(resultHandler)
}
